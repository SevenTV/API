package cosmetics

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"

	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/api/internal/rest/v3/middleware"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

type avatars struct {
	Ctx global.Context
}

func newAvatars(gCtx global.Context) rest.Route {
	return &avatars{gCtx}
}

// Config implements rest.Route
func (r *avatars) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:      "/avatars",
		Method:   rest.GET,
		Children: []rest.Route{},
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 600, nil),
		},
	}
}

// Handler implements rest.Route
func (r *avatars) Handler(ctx *rest.Ctx) errors.APIError {
	mapTo := utils.B2S(ctx.QueryArgs().Peek("map_to"))
	if mapTo == "" || utils.Contains([]string{}, mapTo) {
		mapTo = "hash"
	}

	pipeline := mongo.Pipeline{
		{{
			Key: "$match",
			Value: bson.M{"avatar_id": bson.M{
				"$exists": true,
				"$not":    bson.M{"$in": bson.A{"", nil}},
			}},
		}},
		{{
			Key: "$group",
			Value: bson.M{
				"_id":   nil,
				"users": bson.M{"$push": "$$ROOT"},
			},
		}},
		// Lookup entitlements
		{{
			Key: "$lookup",
			Value: mongo.Lookup{
				From:         mongo.CollectionNameEntitlements,
				LocalField:   "users._id",
				ForeignField: "user_id",
				As:           "role_entitlements",
			},
		}},
		{{
			Key: "$set",
			Value: bson.M{
				"role_entitlements": bson.M{"$filter": bson.M{
					"input": "$role_entitlements",
					"as":    "e",
					"cond": bson.M{
						"$eq": bson.A{"$$e.kind", structures.EntitlementKindRole},
					},
				}},
			},
		}},
	}
	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).Aggregate(ctx, pipeline)
	if err != nil {
		zap.S().Errorw("mongo, failed to spawn aggregation for user avatars",
			"error", err,
		)
		return errors.ErrInternalServerError().SetDetail(err.Error())
	}
	v := &aggregatedAvatarsResult{}
	cur.Next(ctx)
	if err = cur.Decode(v); err != nil {
		zap.S().Errorw("mongo, failed to decode aggregated avatars data",
			"error", err,
		)
		return errors.ErrInternalServerError().SetDetail(err.Error())
	}

	// Compose the result
	qb := r.Ctx.Inst().Query.NewBinder(ctx)
	userMap, err := qb.MapUsers(v.Users, v.RoleEntilements...)
	if err != nil {
		return errors.ErrInternalServerError().SetDetail(err.Error())
	}
	result := make(map[string]string)
	for _, u := range userMap {
		if !u.HasPermission(structures.RolePermissionFeatureProfilePictureAnimation) {
			continue
		}
		// Get user's twitch connction
		tw, _, err := u.Connections.Twitch()
		if err != nil {
			continue // skip if no twitch connection
		}

		key := ""
		switch mapTo {
		case "hash":
			key = hashAvatarURL(tw.Data.ProfileImageURL)
		case "object_id":
			key = u.ID.Hex()
		case "login":
			key = u.Username
		default:
			continue
		}
		result[key] = fmt.Sprintf("https://%s/pp/%s/%s", r.Ctx.Config().CdnURL, u.ID.Hex(), u.AvatarID)
	}

	return ctx.JSON(rest.OK, result)
}

var avatarSizeRegex = regexp.MustCompile("([0-9]{2,3})x([0-9]{2,3})")

func hashAvatarURL(u string) string {
	u = avatarSizeRegex.ReplaceAllString(u, "300x300")
	hasher := sha256.New()
	hasher.Write(utils.S2B(u))
	return hex.EncodeToString(hasher.Sum(nil))
}

type aggregatedAvatarsResult struct {
	Users           []structures.User                  `bson:"users"`
	RoleEntilements []structures.Entitlement[bson.Raw] `bson:"role_entitlements"`
}
