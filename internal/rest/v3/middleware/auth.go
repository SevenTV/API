package middleware

import (
	"strings"
	"time"

	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/common/auth"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/query"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Auth(gCtx global.Context) rest.Middleware {
	return func(ctx *rest.Ctx) rest.APIError {
		// Parse token from header
		h := utils.B2S(ctx.Request.Header.Peek("Authorization"))
		s := strings.Split(h, "Bearer ")

		if len(s) != 2 {
			return errors.ErrUnauthorized().SetFields(errors.Fields{"message": "Bad Authorization Header"})
		}

		t := s[1]

		// Verify the token
		claims := &auth.JWTClaimUser{}

		_, err := auth.VerifyJWT(gCtx.Config().Credentials.JWTSecret, strings.Split(t, "."), claims)
		if err != nil {
			return errors.ErrUnauthorized().SetFields(errors.Fields{"message": err.Error()})
		}

		// User ID from parsed token
		if claims.UserID == "" {
			return errors.ErrUnauthorized().SetFields(errors.Fields{"message": "Bad Token"})
		}

		userID, err := primitive.ObjectIDFromHex(claims.UserID)
		if err != nil {
			return errors.ErrUnauthorized().SetFields(errors.Fields{"message": err.Error()})
		}

		user, err := gCtx.Inst().Query.Users(ctx, bson.M{"_id": userID}).First()
		if err != nil {
			return errors.From(err)
		}

		if user.TokenVersion != claims.TokenVersion {
			return errors.ErrUnauthorized().SetFields(errors.Fields{"message": "Token Version Mismatch"})
		}

		// Check bans
		bans, err := gCtx.Inst().Query.Bans(ctx, query.BanQueryOptions{
			Filter: bson.M{"effects": bson.M{"$bitsAnySet": structures.BanEffectNoAuth | structures.BanEffectNoPermissions}},
		})
		if err != nil {
			return errors.From(err)
		}

		if ban, noAuth := bans.NoAuth[userID]; noAuth {
			return errors.ErrInsufficientPrivilege().
				SetDetail("You are banned").
				SetFields(errors.Fields{
					"ban_reason":      ban.Reason,
					"ban_expire_date": ban.ExpireAt.Format(time.RFC3339),
				})
		}

		if _, noRights := bans.NoPermissions[userID]; noRights {
			user.Roles = []structures.Role{structures.RevocationRole}
		}

		ctx.SetActor(&user)

		return nil
	}
}
