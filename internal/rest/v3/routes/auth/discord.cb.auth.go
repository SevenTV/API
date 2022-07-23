package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/seventv/api/internal/externalapis"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/common/auth"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/utils"
	"github.com/valyala/fasthttp"
)

type discordCallback struct {
	Ctx global.Context
}

func newDiscordCallback(gctx global.Context) rest.Route {
	return &discordCallback{gctx}
}

func (r *discordCallback) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:        "/discord/callback",
		Method:     rest.GET,
		Children:   []rest.Route{},
		Middleware: []rest.Middleware{},
	}
}

func (r *discordCallback) Handler(ctx *rest.Ctx) rest.APIError {
	// Retrieve state from query
	state := utils.B2S(ctx.QueryArgs().Peek("state"))
	if state == "" {
		return errors.ErrMissingRequiredField().SetFields(errors.Fields{"query": "state"})
	}

	// Retrieve the CSRF token from cookies
	csrfToken := strings.Split(utils.B2S(ctx.Request.Header.Cookie(DISCORD_CSRF_COOKIE_NAME)), ".")
	if len(csrfToken) != 3 {
		return errors.ErrUnauthorized().SetDetail(fmt.Sprintf("Bad State (found %d segments when 3 were expected)", len(csrfToken)))
	}

	// Verify the token
	csrfClaim := &auth.JWTClaimOAuth2CSRF{}

	token, err := auth.VerifyJWT(r.Ctx.Config().Credentials.JWTSecret, csrfToken, csrfClaim)
	if err != nil {
		return errors.ErrUnauthorized().SetDetail(fmt.Sprintf("Invalid State: %s", err.Error()))
	}

	{
		b, err := json.Marshal(token.Claims)
		if err != nil {
			ctx.Log().Errorw("json", "error", err)

			return errors.ErrUnauthorized().SetDetail(fmt.Sprintf("Invalid State: %s", err.Error()))
		}

		if err = json.Unmarshal(b, csrfClaim); err != nil {
			ctx.Log().Errorw("json", "error", err)

			return errors.ErrUnauthorized().SetDetail(fmt.Sprintf("Invalid State: %s", err.Error()))
		}
	}

	// Validate: token date
	if csrfClaim.CreatedAt.Before(time.Now().Add(-time.Minute * 5)) {
		return errors.ErrUnauthorized().SetDetail("Expired State")
	}

	// Check mismatch
	if state != csrfClaim.State {
		return errors.ErrUnauthorized().SetDetail("Mismatched State Value")
	}

	// Remove the CSRF cookie
	cookie := fasthttp.Cookie{}
	cookie.SetKey(DISCORD_CSRF_COOKIE_NAME)
	cookie.SetExpire(time.Now())
	cookie.SetDomain(r.Ctx.Config().Http.Cookie.Domain)
	cookie.SetSecure(r.Ctx.Config().Http.Cookie.Secure)
	cookie.SetHTTPOnly(true)
	ctx.Response.Header.Cookie(&cookie) // We have now validated this request is authentic.

	// OAuth2 auhorization code for granting an access token
	code := utils.B2S(ctx.QueryArgs().Peek("code"))

	// Format querystring for our authenticated request to twitch
	params, err := query.Values(&OAuth2AuthorizationParams{
		ClientID:     r.Ctx.Config().Platforms.Discord.ClientID,
		ClientSecret: r.Ctx.Config().Platforms.Discord.ClientSecret,
		RedirectURI:  r.Ctx.Config().Platforms.Discord.RedirectURI,
		Code:         code,
		GrantType:    "authorization_code",
	})
	if err != nil {
		ctx.Log().Errorw("querystring", "error", err)

		return errors.ErrInternalServerError()
	}

	// Now we will make a request to Discord to retrieve the user data
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/oauth2/token", externalapis.DiscordAPIBase), bytes.NewBuffer(utils.S2B(params.Encode())))
	if err != nil {
		ctx.Log().Errorw("discord", "error", err)

		return errors.ErrInternalServerError().SetDetail("Internal Request to External Provider Failed")
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ctx.Log().Errorw("discord", "error", err)

		return errors.ErrInternalServerError().SetDetail("Internal Request Rejected by External Provider")
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			ctx.Log().Errorw("discord", "error", err)

			return errors.ErrInternalServerError().SetDetail("Internal Request Rejected by External Provider")
		}

		ctx.Log().Errorw("discord, bad resp", "error", string(body), "status", resp.StatusCode, "data", string(body))

		return errors.ErrInternalServerError().SetDetail("Internal Request Rejected by External Provider")
	}

	grant := &OAuth2AuthorizedResponseAlt{}
	if err = externalapis.ReadRequestResponse(resp, grant); err != nil {
		ctx.Log().Errorw("ReadRequestResponse",
			"error", err,
		)

		return errors.ErrInternalServerError().SetDetail("Failed to decode data sent by the External Provider")
	}

	// Retrieve discord user data
	_, err = externalapis.Discord.GetCurrentUser(r.Ctx, grant.AccessToken)
	if err != nil {
		ctx.Log().Errorw("discord, GetCurrentUser", "error", err)

		return errors.ErrInternalServerError().SetDetail("Couldn't fetch data from the external provider")
	}

	// TODO: add the user data to db and complete the connection

	return nil
}
