package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/common/auth"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/utils"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

const DISCORD_CSRF_COOKIE_NAME = "csrf_token_di"

var discordScopes = []string{
	"identify",
}

type discord struct {
	Ctx global.Context
}

func newDiscord(gctx global.Context) rest.Route {
	return &discord{gctx}
}

func (r *discord) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:        "/discord",
		Method:     rest.GET,
		Children:   []rest.Route{},
		Middleware: []rest.Middleware{},
	}
}

func (r *discord) Handler(ctx *rest.Ctx) rest.APIError {
	// Generate a randomized value for a CSRF token
	csrfValue, err := utils.GenerateRandomString(64)
	if err != nil {
		ctx.Log().Errorw("csrf, random bytes", "error", err)

		return errors.ErrInternalServerError()
	}

	// Sign JWT for CSRF
	csrfToken, err := auth.SignJWT(r.Ctx.Config().Credentials.JWTSecret, auth.JWTClaimOAuth2CSRF{
		State:     csrfValue,
		CreatedAt: time.Now(),
	})
	if err != nil {
		ctx.Log().Errorw("csrf, jwt", "error", err)

		return errors.ErrInternalServerError()
	}

	// Set cookie
	cookie := fasthttp.Cookie{}
	cookie.SetKey(DISCORD_CSRF_COOKIE_NAME)
	cookie.SetValue(csrfToken)
	cookie.SetExpire(time.Now().Add(time.Minute * 5))
	cookie.SetHTTPOnly(true)
	cookie.SetDomain(r.Ctx.Config().Http.Cookie.Domain)
	cookie.SetSecure(r.Ctx.Config().Http.Cookie.Secure)
	ctx.Response.Header.SetCookie(&cookie)

	// Format querystring options for the redirection URL
	params, err := query.Values(&OAuth2URLParams{
		ClientID:     r.Ctx.Config().Platforms.Discord.ClientID,
		RedirectURI:  r.Ctx.Config().Platforms.Discord.RedirectURI,
		ResponseType: "code",
		Scope:        strings.Join(discordScopes, " "),
		State:        csrfValue,
	})
	if err != nil {
		zap.S().Errorw("querystring",
			"error", err,
		)

		return errors.ErrInternalServerError()
	}

	ctx.Redirect(fmt.Sprintf("https://discord.com/api/oauth2/authorize?%s", params.Encode()), int(rest.Found))

	return nil
}
