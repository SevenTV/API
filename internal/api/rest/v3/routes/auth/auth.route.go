package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/seventv/api/data/events"
	"github.com/seventv/api/internal/api/rest/middleware"
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
		Middleware: []rest.Middleware{
			bindCurrentAccessToken,
			middleware.Auth(r.gctx, false),
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
		stateCookie, claim, err := r.gctx.Inst().Auth.ValidateCSRF(state, utils.B2S(ctx.Request.Header.Cookie(string(auth.COOKIE_CSRF))))
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
			"$or": bson.A{
				bson.M{"connections.id": id},
				bson.M{"_id": claim.Bind},
			},
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
			ub.User = structures.User{
				ID:           primitive.NewObjectIDFromTimestamp(time.Now()),
				TokenVersion: 1.0,
				RoleIDs:      []primitive.ObjectID{},
				Editors:      []structures.UserEditor{},
				Connections:  []structures.UserConnection[bson.Raw]{formatUserConnection(id, platform, b, grant)},
				State: structures.UserState{
					LastLoginDate: time.Now(),
					LastVisitDate: time.Now(),
				},
			}

			ub.User.SetDiscriminator("")
			ub.User.InferUsername()

			if _, err := r.gctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).InsertOne(ctx, ub.User); err != nil {
				ctx.Log().Errorw("auth, insert user", "error", err)

				return errors.ErrInternalServerError()
			}
		} else {
			// User already exists; update their data
			didUpdate := ub.User.UpdateConnectionData(id, b)
			if !didUpdate { // if the connection didn't exist, create it
				// Check that the connection isn't already owned by another user
				count, err := r.gctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).CountDocuments(ctx, bson.M{
					"connections.id": id,
				})
				if err != nil {
					ctx.Log().Errorw("auth, failed to check if connection is bound to another location")

					return errors.ErrInternalServerError()
				}

				if count > 0 {
					return errors.ErrInsufficientPrivilege().SetDetail("This connection is already bound to another user")
				}

				con := formatUserConnection(id, platform, b, grant)
				ub.AddConnection(con)

				// eventapi: dispatch the connection create event
				_ = r.gctx.Inst().Events.Dispatch(ctx, events.EventTypeUpdateUser, events.ChangeMap{
					ID:    ub.User.ID,
					Kind:  structures.ObjectKindUser,
					Actor: r.gctx.Inst().Modelizer.User(ub.User).ToPartial(),
					Pushed: []events.ChangeField{{
						Key:   "connections",
						Index: utils.PointerOf(int32(len(ub.User.Connections) - 1)),
						Type:  events.ChangeFieldTypeObject,
						Value: r.gctx.Inst().Modelizer.UserConnection(con),
					}},
				}, events.EventCondition{"object_id": ub.User.ID.Hex()})
			}

			t := time.Now()
			ub.User.State.LastLoginDate = t
			ub.Update.Set("state.last_login_at", t)

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
		ctx.Redirect(fmt.Sprintf("%s/auth/callback?platform=%s&token=%s", r.gctx.Config().WebsiteURL, platform, token), http.StatusFound)
	} else { // This is a request for an authorization URL
		actor, _ := ctx.GetActor()

		// Get csrf token
		csrfValue, csrfToken, err := r.gctx.Inst().Auth.CreateCSRFToken(actor.ID)
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

func formatUserConnection(id string, platform structures.UserConnectionPlatform, b []byte, grant auth.OAuth2AuthorizedResponse) structures.UserConnection[bson.Raw] {
	return structures.UserConnection[bson.Raw]{
		ID:         id,
		Platform:   platform,
		LinkedAt:   time.Now(),
		EmoteSlots: 600,
		Data:       b,
		Grant: &structures.UserConnectionGrant{
			AccessToken:  grant.AccessToken,
			RefreshToken: grant.RefreshToken,
			Scope:        []string{},
			ExpiresAt:    time.Now().Add(time.Duration(grant.ExpiresIn) * time.Second),
		},
	}
}

func bindCurrentAccessToken(ctx *rest.Ctx) rest.APIError {
	tok := utils.B2S(ctx.QueryArgs().Peek("token"))
	if tok == "" {
		return nil
	}

	req := &ctx.Request
	req.Header.Set("Authorization", "Bearer "+tok)

	return nil
}
