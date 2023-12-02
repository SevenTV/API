package middleware

import (
	"strings"
	"time"

	"github.com/seventv/api/data/query"
	"github.com/seventv/api/internal/constant"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/svc/auth"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func Auth(gctx global.Context) Middleware {
	return func(ctx *fasthttp.RequestCtx) errors.APIError {
		token := utils.B2S(ctx.Request.Header.Cookie(string(auth.COOKIE_AUTH)))
		if token == "" {
			// no token from cookie
			// parse token from header
			h := utils.B2S(ctx.Request.Header.Peek("Authorization"))
			if len(h) == 0 {
				return nil
			}

			s := strings.Split(h, "Bearer ")
			if len(s) != 2 {
				return errors.ErrUnauthorized().SetDetail("Bad Authorization Header")
			}

			token = s[1]
		}

		user, err := DoAuth(gctx, token)
		if err != nil {
			return err
		}

		// Write current IP
		clientIP := ""
		switch v := ctx.UserValue(constant.ClientIP).(type) {
		case string:
			clientIP = v
		}

		ipDidChange := clientIP != "" && user.State.ClientIP != clientIP
		lastVisitDidChange := user.State.LastVisitDate.Before(time.Now().Add(-time.Hour * 1))
		usernameDidChange := false
		currentUsername := user.Username

		// Check for username change
		// Find primary user account
		if len(user.Connections) > 0 {
			conn := user.Connections[0]
			connUsername, connDisplayName := conn.Username()

			usernameDidChange = connUsername != user.Username || connDisplayName != user.DisplayName
			if usernameDidChange {
				user.Username, user.DisplayName = connUsername, connDisplayName

				user.SetDiscriminator("")
				user.InferUsername()
			}
		}

		if ipDidChange || lastVisitDidChange || usernameDidChange {
			user.State.ClientIP = clientIP

			m := bson.M{
				"state.client_ip":     clientIP,
				"state.last_visit_at": time.Now(),
			}

			if usernameDidChange {
				m["username"] = user.Username
				m["display_name"] = user.DisplayName
				m["state.username_changed_at"] = time.Now()
				m["state.former_username"] = currentUsername
			}

			if _, err := gctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).UpdateOne(gctx, bson.M{
				"_id": user.ID,
			}, bson.M{
				"$set": m,
			}); err != nil {
				zap.S().Errorw("failed to update user client IP", "error", err)
			}
		}

		ctx.SetUserValue(constant.UserKey, user)
		ctx.Response.Header.Set("X-Actor-ID", user.ID.Hex())

		return nil
	}
}

func DoAuth(ctx global.Context, t string) (structures.User, errors.APIError) {
	// Verify the token
	claims := &auth.JWTClaimUser{}

	user := structures.User{}

	_, err := ctx.Inst().Auth.VerifyJWT(strings.Split(t, "."), claims)
	if err != nil {
		return user, errors.ErrUnauthorized().SetDetail(err.Error())
	}

	// User ID from parsed token
	if claims.UserID == "" {
		return user, errors.ErrUnauthorized().SetDetail("Bad Token")
	}

	userID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		return user, errors.ErrUnauthorized().SetDetail(err.Error())
	}

	user, err = ctx.Inst().Query.Users(ctx, bson.M{"_id": userID}).First()
	if err != nil {
		return user, errors.From(err)
	}

	if user.TokenVersion != claims.TokenVersion {
		return user, errors.ErrUnauthorized().SetDetail("Token Version Mismatch")
	}

	// Check bans
	bans, err := ctx.Inst().Query.Bans(ctx, query.BanQueryOptions{
		Filter: bson.M{"effects": bson.M{"$bitsAnySet": structures.BanEffectNoAuth | structures.BanEffectNoPermissions}},
	})
	if err != nil {
		return user, errors.ErrInternalServerError().SetDetail("Failed")
	}

	if _, noRights := bans.NoPermissions[userID]; noRights {
		user.Roles = []structures.Role{structures.RevocationRole}
	}

	if ban, noAuth := bans.NoAuth[userID]; noAuth {
		user.Bans = append(user.Bans, ban)

		return user, errors.ErrBanned().SetDetail(ban.Reason).SetFields(errors.Fields{
			"ban": map[string]string{
				"reason":    ban.Reason,
				"expire_at": ban.ExpireAt.Format(time.RFC3339),
			},
		})
	}

	return user, nil
}
