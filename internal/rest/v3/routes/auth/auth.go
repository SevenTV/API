package auth

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/common/auth"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/utils"
	"github.com/valyala/fasthttp"
)

type Route struct {
	Ctx global.Context
}

func New(gCtx global.Context) rest.Route {
	return &Route{gCtx}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/auth",
		Method: rest.GET,
		Children: []rest.Route{
			newTwitch(r.Ctx),
			newTwitchCallback(r.Ctx),
			newDiscord(r.Ctx),
			newDiscordCallback(r.Ctx),
		},
		Middleware: []rest.Middleware{},
	}
}

func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	return errors.ErrInvalidRequest().WithHTTPStatus(int(rest.SeeOther)).SetDetail("Use OAuth2 routes")
}

type OAuth2URLParams struct {
	ClientID     string `url:"client_id"`
	RedirectURI  string `url:"redirect_uri"`
	ResponseType string `url:"response_type"`
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

type OAuth2AuthorizedResponse[S []string | string] struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Scope        S      `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
}

// handleOAuthState verifies the csrf token of an oauth2 authorization flow
func handleOAuthState(gctx global.Context, ctx *rest.Ctx) (*auth.JWTClaimOAuth2CSRF, error) {
	// Retrieve state from query
	state := utils.B2S(ctx.QueryArgs().Peek("state"))
	if state == "" {
		return nil, errors.ErrMissingRequiredField().SetFields(errors.Fields{"query": "state"})
	}

	// Retrieve the CSRF token from cookies
	csrfToken := strings.Split(utils.B2S(ctx.Request.Header.Cookie(DISCORD_CSRF_COOKIE_NAME)), ".")
	if len(csrfToken) != 3 {
		return nil, errors.ErrUnauthorized().SetDetail(fmt.Sprintf("Bad State (found %d segments when 3 were expected)", len(csrfToken)))
	}

	// Verify the token
	csrfClaim := &auth.JWTClaimOAuth2CSRF{}

	token, err := auth.VerifyJWT(gctx.Config().Credentials.JWTSecret, csrfToken, csrfClaim)
	if err != nil {
		return nil, errors.ErrUnauthorized().SetDetail(fmt.Sprintf("Invalid State: %s", err.Error()))
	}

	{
		b, err := json.Marshal(token.Claims)
		if err != nil {
			ctx.Log().Errorw("json", "error", err)

			return nil, errors.ErrUnauthorized().SetDetail(fmt.Sprintf("Invalid State: %s", err.Error()))
		}

		if err = json.Unmarshal(b, csrfClaim); err != nil {
			ctx.Log().Errorw("json", "error", err)

			return nil, errors.ErrUnauthorized().SetDetail(fmt.Sprintf("Invalid State: %s", err.Error()))
		}
	}

	// Validate: token date
	if csrfClaim.CreatedAt.Before(time.Now().Add(-time.Minute * 5)) {
		return nil, errors.ErrUnauthorized().SetDetail("Expired State")
	}

	// Check mismatch
	if state != csrfClaim.State {
		return nil, errors.ErrUnauthorized().SetDetail("Mismatched State Value")
	}

	// Remove the CSRF cookie
	cookie := fasthttp.Cookie{}
	cookie.SetKey(DISCORD_CSRF_COOKIE_NAME)
	cookie.SetExpire(time.Now())
	cookie.SetDomain(gctx.Config().Http.Cookie.Domain)
	cookie.SetSecure(gctx.Config().Http.Cookie.Secure)
	cookie.SetHTTPOnly(true)
	ctx.Response.Header.Cookie(&cookie) // We have now validated this request is authentic.

	return csrfClaim, nil
}
