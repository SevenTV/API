package cosmetics

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/middleware"
	"github.com/seventv/api/internal/rest/rest"
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

type aggregatedAvatarsResult struct {
	Users           []structures.User                  `bson:"users"`
	RoleEntilements []structures.Entitlement[bson.Raw] `bson:"role_entitlements"`
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
			middleware.SetCacheControl(r.Ctx, 600, []string{"s-maxage=600"}),
		},
	}
}

// Handler implements rest.Route
func (r *avatars) Handler(ctx *rest.Ctx) errors.APIError {
	mxKey := r.Ctx.Inst().Redis.ComposeKey("api-rest", "lock", "cosmetics-v2:avatars")
	mx := r.Ctx.Inst().Redis.Mutex(mxKey, time.Second*30)

	if err := mx.Lock(); err != nil {
		ctx.Log().Errorw("Failed to acquire lock for cosmetics v2 (avatars)", "error", err)

		return errors.ErrInternalServerError()
	}

	defer func() {
		if _, err := mx.Unlock(); err != nil {
			ctx.Log().Errorw("Failed to release lock for cosmetics v2 (avatars)", "error", err)
		}
	}()

	mapTo := utils.B2S(ctx.QueryArgs().Peek("map_to"))
	if mapTo == "" || utils.Contains([]string{}, mapTo) {
		mapTo = "hash"
	}

	// Compose cache key
	cacheKey := r.Ctx.Inst().Redis.ComposeKey("rest", fmt.Sprintf("cache:cosmetics:avatars:%s", mapTo))

	result := make(map[string]string)

	// Return existing cache?
	d, err := r.Ctx.Inst().Redis.Get(ctx, cacheKey)
	if err == nil && d != "" {
		if err := json.Unmarshal(utils.S2B(d), &result); err != nil {
			zap.S().Errorw("failed to return cache of /v2/cosmetics",
				"error", err,
			)
		}

		return ctx.JSON(rest.OK, result)
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

	for _, u := range userMap {
		if !u.HasPermission(structures.RolePermissionFeatureProfilePictureAnimation) {
			continue
		}
		// Get user's twitch connction
		tw, _, err := u.Connections.Twitch()
		if err != nil {
			continue // skip if no twitch connection
		}

		if strings.HasPrefix(tw.Data.ProfileImageURL, "https://static-cdn.jtvnw.net/user-default-pictures-uv") {
			continue
		}

		key := ""

		switch mapTo {
		case "hash":
			key = hashAvatarURL(tw.Data.ProfileImageURL)
		case "object_id":
			key = u.ID.Hex()
		case "login":
			key = tw.Data.Login
		default:
			continue
		}

		result[key] = fmt.Sprintf("https://%s/%s", r.Ctx.Config().CdnURL, r.Ctx.Inst().S3.ComposeKey("pp", u.ID.Hex(), u.AvatarID))
	}

	b, _ := json.Marshal(result)
	if err := r.Ctx.Inst().Redis.SetEX(ctx, cacheKey, utils.B2S(b), 10*time.Minute); err != nil {
		zap.S().Errorw("couldn't save cosmetics response to redis cache",
			"map_to", mapTo,
		)
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
