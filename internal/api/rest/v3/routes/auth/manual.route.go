package auth

import (
	"encoding/hex"
	"strings"
	"time"

	"github.com/seventv/api/internal/api/rest/middleware"
	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/svc/auth"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
)

type verifyRoute struct {
	gctx global.Context
}

func newManual(gctx global.Context) rest.Route {
	return &verifyRoute{gctx}
}

func (r *verifyRoute) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:      "/manual",
		Method:   rest.GET,
		Children: []rest.Route{},
		Middleware: []rest.Middleware{
			bindCurrentAccessToken,
			middleware.Auth(r.gctx, false),
		},
	}
}

func (r *verifyRoute) Handler(ctx *rest.Ctx) errors.APIError {
	verify := ctx.QueryArgs().GetBool("verify")

	platform := structures.UserConnectionPlatform(strings.ToUpper(utils.B2S(ctx.QueryArgs().Peek("platform"))))
	if !platform.Supported() {
		return errors.ErrInvalidRequest().SetDetail("Unsupported Account Provider")
	}

	accountID := utils.B2S(ctx.QueryArgs().Peek("id"))
	if accountID == "" {
		return errors.ErrMissingRequiredField().SetFields(errors.Fields{"query": "id"})
	}

	codeKey := r.gctx.Inst().Redis.ComposeKey(
		"api",
		"manual-auth", strings.ToLower(string(platform)),
		accountID,
	)

	// request for a verification code
	if !verify {
		b, err := utils.GenerateRandomBytes(16)
		if err != nil {
			return errors.ErrInternalServerError().SetDetail(err.Error())
		}

		code := hex.EncodeToString(b)

		if err := r.gctx.Inst().Redis.SetEX(ctx, codeKey, code, 5*time.Minute); err != nil {
			ctx.Log().Errorw("failed to set redis key", "error", err)

			return errors.ErrInternalServerError()
		}

		_, _ = ctx.WriteString(code)

		return nil
	} else {
		// request to verify the code
		id, userData, err := r.gctx.Inst().Auth.UserData(structures.UserConnectionPlatformKick, accountID)
		if err != nil {
			ctx.Log().Errorw("failed to get user data", "error", err)

			return errors.ErrInternalServerError().SetDetail(err.Error())
		}

		ub := structures.NewUserBuilder(structures.User{})

		actor, actorOk := ctx.GetActor()

		// Query existing user?
		if actorOk {
			if err := r.gctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).FindOne(ctx, bson.M{
				"$or": bson.A{
					bson.M{"connections.id": id},
					bson.M{"_id": actor.ID},
				},
			}).Decode(&ub.User); err != nil && err != mongo.ErrNoDocuments {
				ctx.Log().Errorw("auth, find user", "error", err)

				return errors.ErrInternalServerError()
			}
		}

		// Convert JSON to BSON
		var connData bson.M
		err = bson.UnmarshalExtJSON(userData, true, &connData)

		if err != nil {
			ctx.Log().Errorw("auth, convert bson document", "error", err)

			return errors.ErrInternalServerError()
		}

		// Marshal into raw bson document
		b, err := bson.Marshal(connData)
		if err != nil {
			ctx.Log().Errorw("auth, encode bson document", "error", err)

			return errors.ErrInternalServerError()
		}

		// Verify the code
		code, err := r.gctx.Inst().Redis.Get(ctx, codeKey)
		if err != nil {
			ctx.Log().Errorw("failed to get redis key", "error", err)

			return errors.ErrInternalServerError()
		}

		// Verify the code
		if !strings.Contains(utils.B2S(userData), string(code)) {
			return errors.ErrUnauthorized().SetDetail("Invalid Code")
		}

		if err = setupUser(r.gctx, ctx, b, ub, id, platform, auth.OAuth2AuthorizedResponse{}); err != nil {
			return errors.From(err)
		}

		token, _, err := setupToken(r.gctx, ub)
		if err != nil {
			return errors.From(err)
		}

		ctx.Response.Header.Set("X-Access-Token", token)
	}

	return nil
}
