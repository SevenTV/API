package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-querystring/query"
	"github.com/nicklaw5/helix"
	"github.com/seventv/api/internal/configure"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type Authorizer interface {
	SignJWT(secret string, claim jwt.Claims) (string, error)
	VerifyJWT(token []string, out jwt.Claims) (*jwt.Token, error)
	CreateCSRFToken(targetID primitive.ObjectID) (value, token string, err error)
	CreateAccessToken(targetID primitive.ObjectID, version float64) (string, time.Time, error)
	ValidateCSRF(state string, cookieData string) (*fasthttp.Cookie, *JWTClaimOAuth2CSRF, error)
	Cookie(key, token string, duration time.Duration) *fasthttp.Cookie
	QueryValues(provider structures.UserConnectionPlatform, csrfToken string) (url.Values, error)
	ExchangeCode(ctx context.Context, provider structures.UserConnectionPlatform, code string) (OAuth2AuthorizedResponse, error)
	TwichUserData(grant string) (string, []byte, error)
	DiscordUserData(grant string) (string, []byte, error)
	UserData(provider structures.UserConnectionPlatform, token string) (id string, b []byte, err error)
	LocateIP(ctx context.Context, ip string) (GeoIPResult, error)
}

type authorizer struct {
	JWTSecret string
	Domain    string
	Secure    bool
	Redis     redis.Instance
	Config    configure.PlatformConfig

	helixFactory   func() (*helix.Client, error)
	discordFactory func(token string) (*discordgo.Session, error)
	kickClient     *http.Client
}

const (
	COOKIE_CSRF = "seventv-csrf"
	COOKIE_AUTH = "seventv-auth"
)

func New(ctx context.Context, opt AuthorizerOptions) Authorizer {
	a := &authorizer{
		JWTSecret: opt.JWTSecret,
		Domain:    opt.Domain,
		Secure:    opt.Secure,
		Config:    opt.Config,
		Redis:     opt.Redis,
	}

	a.helixFactory = func() (*helix.Client, error) {
		return helix.NewClient(&helix.Options{
			ClientID:     a.Config.Twitch.ClientID,
			ClientSecret: a.Config.Twitch.ClientSecret,
		})
	}

	a.discordFactory = func(token string) (*discordgo.Session, error) {
		return discordgo.New("Bearer " + token)
	}

	if a.Config.Kick.ChallengeToken != "" {
		a.kickClient = newKickClient(ctx, a.Config.Kick.ChallengeToken)
	}

	return a
}

type AuthorizerOptions struct {
	JWTSecret string
	Domain    string
	Secure    bool
	Config    configure.PlatformConfig
	Redis     redis.Instance
}

// CreateCSRFToken creates a CSRF token
func (a *authorizer) CreateCSRFToken(targetID primitive.ObjectID) (value, token string, err error) {
	// Generate a randomized value for a CSRF token
	value, err = utils.GenerateRandomString(64)
	if err != nil {
		zap.S().Errorw("csrf, random bytes",
			"error", err,
			"target_id", targetID,
		)

		return "", "", err
	}

	token, err = a.SignJWT(a.JWTSecret, &JWTClaimOAuth2CSRF{
		State:     value,
		CreatedAt: time.Now(),
		Bind:      targetID,
	})
	if err != nil {
		zap.S().Errorw("csrf, sign",
			"error", err,
			"target_id", targetID,
		)

		return "", "", err
	}

	return value, token, nil
}

func (a *authorizer) CreateAccessToken(targetID primitive.ObjectID, version float64) (string, time.Time, error) {
	expireAt := time.Now().Add(time.Hour * 24 * 90)

	token, err := a.SignJWT(a.JWTSecret, &JWTClaimUser{
		UserID:       targetID.Hex(),
		TokenVersion: version,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "seventv-api",
			ExpiresAt: &jwt.NumericDate{Time: expireAt}, // 90 days
			NotBefore: &jwt.NumericDate{Time: time.Now()},
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
		},
	})
	if err != nil {
		zap.S().Errorw("access_token, sign",
			"error", err,
			"target_id", targetID,
		)

		return "", time.Time{}, err
	}

	return token, expireAt, nil
}

func (a *authorizer) ValidateCSRF(state string, cookieData string) (*fasthttp.Cookie, *JWTClaimOAuth2CSRF, error) {
	// Retrieve the CSRF token from cookies
	csrfToken := strings.Split(cookieData, ".")

	if len(csrfToken) != 3 {
		return nil, nil, fmt.Errorf("bad state (found %d segments when 3 were expected)", len(csrfToken))
	}

	// Verify the token
	csrfClaim := &JWTClaimOAuth2CSRF{}

	token, err := a.VerifyJWT(csrfToken, csrfClaim)
	if err != nil {
		return nil, csrfClaim, fmt.Errorf("invalid state: %s", err.Error())
	}

	{
		b, err := json.Marshal(token.Claims)
		if err != nil {
			return nil, csrfClaim, fmt.Errorf("invalid state: %s", err.Error())
		}

		if err = json.Unmarshal(b, csrfClaim); err != nil {
			return nil, csrfClaim, fmt.Errorf("invalid state: %s", err.Error())
		}
	}

	// Validate: token date
	if csrfClaim.CreatedAt.Before(time.Now().Add(-time.Minute * 5)) {
		return nil, csrfClaim, fmt.Errorf("expired state")
	}

	// Check mismatch
	if state != csrfClaim.State {
		return nil, csrfClaim, fmt.Errorf("mismatched state value")
	}

	// Udate the CSRF cookie (immediate expire)
	cookie := a.Cookie(string(COOKIE_CSRF), "", 0)

	return cookie, csrfClaim, nil
}

// Cookie returns a cookie
func (a *authorizer) Cookie(key, token string, duration time.Duration) *fasthttp.Cookie {
	cookie := &fasthttp.Cookie{}
	cookie.SetKey(key)
	cookie.SetValue(token)
	cookie.SetExpire(time.Now().Add(duration))
	cookie.SetHTTPOnly(true)
	cookie.SetDomain(a.Domain)
	cookie.SetPath("/")
	cookie.SetSecure(a.Secure)
	cookie.SetSameSite(fasthttp.CookieSameSiteNoneMode)

	return cookie
}

func (a *authorizer) oauthCredentials(provider structures.UserConnectionPlatform) (scope, clientID, clientSecret, redirectURI string) {
	switch provider {
	case structures.UserConnectionPlatformTwitch:
		scope = strings.Join(twitchScopes, " ")
		clientID = a.Config.Twitch.ClientID
		clientSecret = a.Config.Twitch.ClientSecret
		redirectURI = a.Config.Twitch.RedirectURI
	case structures.UserConnectionPlatformDiscord:
		scope = strings.Join(discordScopes, " ")
		clientID = a.Config.Discord.ClientID
		clientSecret = a.Config.Discord.ClientSecret
		redirectURI = a.Config.Discord.RedirectURI

	default:
		zap.S().Warnw("QueryValues(), unknown provider", "provider", provider)
	}

	return scope, clientID, clientSecret, redirectURI
}

// QueryValues returns the query values for the OAuth2 Authorize Request
func (a *authorizer) QueryValues(provider structures.UserConnectionPlatform, csrfToken string) (url.Values, error) {
	scope, clientID, _, redirectURI := a.oauthCredentials(provider)

	// Format querystring options for the redirection URL
	params, err := query.Values(&OAuth2URLParams{
		ClientID:     clientID,
		RedirectURI:  redirectURI,
		ResponseType: "code",
		Scope:        scope,
		State:        csrfToken,
	})
	if err != nil {
		zap.S().Errorw("querystring",
			"error", err,
		)

		return nil, err
	}

	return params, nil
}

// ExchangeCode exchanges an OAuth2 code for an access token
func (a *authorizer) ExchangeCode(ctx context.Context, provider structures.UserConnectionPlatform, code string) (OAuth2AuthorizedResponse, error) {
	_, clientID, clientSecret, redirectURI := a.oauthCredentials(provider)

	grant := OAuth2AuthorizedResponse{}

	var (
		req *http.Request
		err error
	)

	// Format querystring for our authenticated request
	params, err := query.Values(&OAuth2AuthorizationParams{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		Code:         code,
		GrantType:    "authorization_code",
	})
	if err != nil {
		zap.S().Errorw("querystring",
			"error", err,
		)

		return grant, errors.ErrInternalServerError()
	}

	switch provider {
	case structures.UserConnectionPlatformDiscord:
		// Prepare a HTTP request to the provider to convert code to acccess token
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, provider.TokenURL(), strings.NewReader(params.Encode()))
		if err != nil {
			zap.S().Errorw("auth(ExchangeCode)",
				"error", err,
				"provider", provider,
			)

			return grant, errors.ErrInternalServerError()
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	case structures.UserConnectionPlatformTwitch, structures.UserConnectionPlatformYouTube:
		req, err = http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s?%s", provider.TokenURL(), params.Encode()), nil)
		if err != nil {
			zap.S().Errorw("auth(ExchangeCode)",
				"error", err,
				"provider", provider,
			)

			return grant, err
		}
	case structures.UserConnectionPlatformKick:
		break
	}

	if err != nil {
		zap.S().Errorw("auth(ExchangeCode)",
			"error", err,
			"provider", provider,
		)

		return grant, err
	}

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		zap.S().Errorw("auth(ExchangeCode)",
			"error", err,
		)

		return grant, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			zap.S().Errorw("auth(ExchangeCode)",
				"error", err,
			)

			return grant, err
		}

		err = fmt.Errorf("bad resp from provider: %d - %s", resp.StatusCode, body)

		return grant, err
	}

	// Decode the grant response and return the access token
	b, err := io.ReadAll(resp.Body)

	if err != nil {
		return grant, err
	}

	if err = json.Unmarshal(b, &grant); err != nil {
		return grant, err
	}

	return grant, nil
}

type OAuth2URLParams struct {
	ClientID     string `url:"client_id"`
	RedirectURI  string `url:"redirect_uri"`
	ResponseType string `url:"response_type,omitempty"`
	GrantType    string `url:"grant_type,omitempty"`
	Scope        string `url:"scope"`
	State        string `url:"state"`
}

type OAuth2AuthorizationParams struct {
	ClientID     string `url:"client_id" json:"client_id"`
	ClientSecret string `url:"client_secret" json:"client_secret"`
	RedirectURI  string `url:"redirect_uri" json:"redirect_uri"`
	Code         string `url:"code" json:"code"`
	GrantType    string `url:"grant_type" json:"grant_type"`
	Scope        string `url:"scope" json:"scope"`
}

type OAuth2AuthorizedResponse struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}
