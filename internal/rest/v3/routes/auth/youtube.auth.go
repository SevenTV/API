package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/middleware"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/common/auth"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/utils"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

const YOUTUBE_CSRF_COOKIE_NAME = "csrf_token_yt"

var youtubeScopes = []string{
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/youtube.readonly",
}

type youtube struct {
	Ctx global.Context
}

func newYouTube(gCtx global.Context) rest.Route {
	return &youtube{gCtx}
}

func (r *youtube) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:      "/youtube",
		Method:   rest.GET,
		Children: []rest.Route{},
		Middleware: []rest.Middleware{
			bindMiddleware,
			middleware.Auth(r.Ctx, false)},
	}
}

func (r *youtube) Handler(ctx *rest.Ctx) rest.APIError {
	actor, _ := ctx.GetActor()

	// Generate a randomized value for a CSRF token
	csrfValue, err := utils.GenerateRandomString(64)
	if err != nil {
		zap.S().Errorw("csrf, random bytes",
			"error", err,
		)

		return errors.ErrInternalServerError()
	}

	// Sign a JWT with the CSRF bytes
	csrfToken, err := auth.SignJWT(r.Ctx.Config().Credentials.JWTSecret, auth.JWTClaimOAuth2CSRF{
		State:       csrfValue,
		CreatedAt:   time.Now(),
		Bind:        actor.ID,
		OldRedirect: ctx.QueryArgs().GetBool("old"),
	})
	if err != nil {
		zap.S().Errorw("csrf, sign jwt",
			"error", err,
		)

		return errors.ErrInternalServerError()
	}

	// Set the CSRF token as a cookie
	cookie := fasthttp.Cookie{}
	cookie.SetKey(YOUTUBE_CSRF_COOKIE_NAME)
	cookie.SetValue(csrfToken)
	cookie.SetExpire(time.Now().Add(5 * time.Minute))
	cookie.SetHTTPOnly(true)
	cookie.SetDomain(r.Ctx.Config().Http.Cookie.Domain)
	cookie.SetSecure(r.Ctx.Config().Http.Cookie.Secure)
	ctx.Response.Header.SetCookie(&cookie)

	// Format querystring options for the redirection URL
	params, err := query.Values(&OAuth2URLParams{
		ClientID:     r.Ctx.Config().Platforms.YouTube.ClientID,
		RedirectURI:  r.Ctx.Config().Platforms.YouTube.RedirectURI,
		ResponseType: "code",
		Scope:        strings.Join(youtubeScopes, " "),
		State:        csrfValue,
	})
	if err != nil {
		zap.S().Errorw("youtube, query values",
			"error", err,
		)

		return errors.ErrInternalServerError()
	}

	params.Set("prompt", "select_account")

	// Redirect to YouTube OAuth2 URL
	ctx.Redirect(fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?%s", params.Encode()), int(rest.Found))

	return nil
}
