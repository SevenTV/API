package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/svc/auth"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Route struct {
	gctx global.Context
}

func New(gctx global.Context) rest.Route {
	return &Route{gctx}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/auth",
		Method: rest.GET,
		Children: []rest.Route{
			newLogout(r.gctx),
		},
	}
}

func (r *Route) Handler(ctx *rest.Ctx) errors.APIError {
	platform := structures.UserConnectionPlatform(strings.ToUpper(utils.B2S(ctx.QueryArgs().Peek("platform"))))
	if !platform.Supported() {
		return errors.ErrInvalidRequest().SetDetail("Unsupported Account Provider")
	}

	callback := ctx.QueryArgs().GetBool("callback")

	// This is a callback
	if callback {
		// Retrieve state from query
		state := utils.B2S(ctx.QueryArgs().Peek("state"))
		if state == "" {
			return errors.ErrMissingRequiredField().SetFields(errors.Fields{"query": "state"})
		}

		// Validate the state
		stateCookie, err := r.gctx.Inst().Auth.ValidateCSRF(state, utils.B2S(ctx.Request.Header.Cookie(string(auth.COOKIE_CSRF))))
		if err != nil {
			return errors.ErrUnauthorized().SetDetail(err.Error())
		}

		ctx.Response.Header.SetCookie(stateCookie)

		grant, err := r.gctx.Inst().Auth.ExchangeCode(ctx, platform, utils.B2S(ctx.QueryArgs().Peek("code")))
		if err != nil {
			ctx.Log().Warnw("auth, exchange code", "error", err)

			return errors.ErrInvalidRequest().SetDetail(err.Error())
		}

		// Get the user data
		id, b, err := r.gctx.Inst().Auth.UserData(platform, grant.AccessToken)
		if err != nil {
			ctx.Log().Warnw("auth, get user data", "error", err)

			return errors.ErrInvalidRequest().SetDetail(err.Error())
		}

		ub := structures.NewUserBuilder(structures.User{})

		// Query existing user?
		if err := r.gctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).FindOne(ctx, bson.M{
			"connections.id": id,
		}).Decode(&ub.User); err != nil && err != mongo.ErrNoDocuments {
			ctx.Log().Errorw("auth, find user", "error", err)

			return errors.ErrInternalServerError()
		}

		// Convert JSON to BSON
		var connData bson.M
		err = bson.UnmarshalExtJSON(b, true, &connData)

		if err != nil {
			ctx.Log().Errorw("auth, convert bson document", "error", err)

			return errors.ErrInternalServerError()
		}

		// Marshal into raw bson document
		b, err = bson.Marshal(connData)
		if err != nil {
			ctx.Log().Errorw("auth, encode bson document", "error", err)

			return errors.ErrInternalServerError()
		}

		// Create the user
		if ub.User.ID.IsZero() {
			ub.User.SetDiscriminator("")
			ub.User.InferUsername()

			if _, err := r.gctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).InsertOne(ctx, ub.User); err != nil {
				ctx.Log().Errorw("auth, insert user", "error", err)

				return errors.ErrInternalServerError()
			}
		} else {
			// User already exists; update their data
			ub.User.UpdateConnectionData(id, b)

			if _, err := r.gctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).UpdateOne(ctx, bson.M{
				"_id": ub.User.ID,
			}, bson.M{
				"$set": ub.User,
			}); err != nil {
				ctx.Log().Errorw("auth, update user", "error", err)
			}
		}

		// Sign an access token
		token, expiry, err := r.gctx.Inst().Auth.CreateAccessToken(ub.User.ID, ub.User.TokenVersion)
		if err != nil {
			ctx.Log().Errorw("auth, create access token", "error", err)

			return errors.ErrInternalServerError()
		}

		// Set a cookie
		authCookie := r.gctx.Inst().Auth.Cookie(string(auth.COOKIE_AUTH), token, time.Until(expiry))
		ctx.Response.Header.SetCookie(authCookie)

		// Redirect to site
		ctx.Redirect(fmt.Sprintf("%s/auth/callback?platform=%s", r.gctx.Config().WebsiteURL, platform), http.StatusFound)
	} else { // This is a request for an authorization URL
		// Get csrf token
		csrfValue, csrfToken, err := r.gctx.Inst().Auth.CreateCSRFToken(primitive.NilObjectID)
		if err != nil {
			return errors.ErrInternalServerError().SetDetail("csrf failure")
		}

		// Set state cookie
		cookie := r.gctx.Inst().Auth.Cookie(string(auth.COOKIE_CSRF), csrfToken, time.Minute*5)
		ctx.Response.Header.SetCookie(cookie)

		// Format oauth params
		params, err := r.gctx.Inst().Auth.QueryValues(platform, csrfValue)
		if err != nil {
			ctx.Log().Errorw("auth, query values",
				"error", err,
			)

			return errors.ErrInternalServerError().SetDetail("oauth params failure")
		}

		// Redirect to provider
		ctx.Redirect(fmt.Sprintf("%s?%s", platform.AuthorizeURL(), params.Encode()), int(rest.Found))
	}

	return nil
}
