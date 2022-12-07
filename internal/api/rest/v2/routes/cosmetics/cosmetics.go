package cosmetics

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/seventv/api/internal/api/rest/middleware"
	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/api/rest/v2/model"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Route struct {
	Ctx global.Context
	mx  *sync.Mutex
}

func New(gCtx global.Context) rest.Route {
	return &Route{gCtx, &sync.Mutex{}}
}

// Config implements rest.Route
func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/cosmetics",
		Method: rest.GET,
		Children: []rest.Route{
			newAvatars(r.Ctx),
		},
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 600, []string{"s-maxage=600"}),
		},
	}
}

const COSMETICS_CACHE_LIFETIME = time.Minute * 10

// Get Cosmetics
// @Summary Get Cosmetics
// @Description Get all active cosmetics and the users assigned to them
// @Tags cosmetics
// @Param user_identifier query string false "one of 'object_id', 'twitch_id' or 'login'"
// @Produce json
// @Success 200 {object} model.CosmeticsMap
// @Router /cosmetics [get]
func (r *Route) Handler(ctx *rest.Ctx) errors.APIError {
	mxKey := r.Ctx.Inst().Redis.ComposeKey("api-rest", "lock", "cosmetics-v2")
	mx := r.Ctx.Inst().Redis.Mutex(mxKey, time.Second*30)

	if err := mx.Lock(); err != nil {
		ctx.Log().Errorw("Failed to acquire lock for cosmetics v2", "error", err)

		return errors.ErrInternalServerError()
	}

	defer func() {
		// Release the lock if data is fresh
		if _, err := mx.Unlock(); err != nil {
			ctx.Log().Errorw("Failed to release lock for cosmetics v2", "error", err)
		}
	}()

	// identifier type argument
	idType := utils.B2S(ctx.QueryArgs().Peek("user_identifier"))

	if !utils.Contains([]string{"object_id", "twitch_id", "login"}, idType) {
		return errors.ErrInvalidRequest().SetDetail("Query Param 'user_identifier' must be 'object_id', 'twitch_id' or 'login'")
	}

	// Compose cache key
	cacheKey := r.Ctx.Inst().Redis.ComposeKey("rest", fmt.Sprintf("cache:cosmetics:%s", idType))

	// Return existing cache?
	result := &model.CosmeticsMap{}

	d, err := r.Ctx.Inst().Redis.Get(ctx, cacheKey)
	noData := false

	if err == nil && d != "" {
		if err := json.Unmarshal(utils.S2B(d), result); err != nil {
			ctx.Log().Errorw("failed to return cache of /v2/cosmetics",
				"error", err,
			)
		}

		// If the cache is still valid
		timestamp := time.UnixMilli(result.Timestamp)

		// Check if the timestamp is newer than 10 minutes
		if time.Since(timestamp) < COSMETICS_CACHE_LIFETIME {
			return ctx.JSON(rest.OK, result)
		}
	} else {
		noData = true

		if err != redis.Nil {
			ctx.Log().Errorw("redis, failed to get cache of /v2/cosmetics", "error", err)
		}
	}

	// Response channels
	resCh := make(chan cosmeticsChanResult, 1)

	go func() {
		defer func() {
			// Close the response channels
			close(resCh)
		}()

		busyKey := r.Ctx.Inst().Redis.ComposeKey("api-rest", "busy", "cosmetics", idType)
		if val, _ := r.Ctx.Inst().Redis.Get(ctx, busyKey); val == "1" {
			ctx.Log().Errorw("failed to get busy state of generation for cosmetics v2")

			resCh <- cosmeticsChanResult{err: errors.ErrInternalServerError()}

			return
		}

		// Set a "busy" value in redis
		// This will ensure that the data isn't being queried concurrently
		defer func() {
			if _, err := r.Ctx.Inst().Redis.Del(ctx, busyKey); err != nil {
				ctx.Log().Errorw("failed to delete busy key for cosmetics v2", "error", err)
			}
		}()

		_ = r.Ctx.Inst().Redis.SetEX(ctx, busyKey, "1", time.Minute)

		result, err := r.generateCosmeticsData(ctx, idType)
		if err != nil {
			resCh <- cosmeticsChanResult{err: err}
		} else {
			resCh <- cosmeticsChanResult{d: result}

			// Store the result in redis
			b, _ := json.Marshal(result)
			if err := r.Ctx.Inst().Redis.Set(ctx, cacheKey, utils.B2S(b)); err != nil {
				ctx.Log().Errorw("couldn't save cosmetics response to redis cache",
					"id_type", idType,
				)
			}
		}
	}()

	// if we had no pre-existing cache, we must wait for data to be generated
	if noData {
		res := <-resCh

		if res.err != nil {
			return errors.From(res.err)
		}

		result = res.d
	} // if cache existed, we can respond to the request and the data will generate in the background for future requests

	return ctx.JSON(rest.OK, result)
}

type cosmeticsChanResult struct {
	err error
	d   *model.CosmeticsMap
}

func (r *Route) generateCosmeticsData(ctx *rest.Ctx, idType string) (*model.CosmeticsMap, error) {
	// Retrieve all users of badges
	// Find entitlements
	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEntitlements).Aggregate(ctx, mongo.Pipeline{
		{{Key: "$sort", Value: bson.M{"priority": -1}}},
		{{Key: "$match", Value: bson.M{
			"disabled": bson.M{"$not": bson.M{"$eq": true}},
			"kind": bson.M{"$in": []structures.EntitlementKind{
				structures.EntitlementKindRole,
				structures.EntitlementKindBadge,
				structures.EntitlementKindPaint,
			}},
		}}},
	}, options.Aggregate().SetAllowDiskUse(true))
	if err != nil {
		ctx.Log().Errorw("mongo, failed to spawn cosmetic entitlements aggregation", "error", err)
		return nil, errors.ErrInternalServerError()
	}

	// Decode data
	entitlements := []structures.Entitlement[bson.Raw]{}
	if err = cur.All(ctx, &entitlements); err != nil {
		ctx.Log().Errorw("mongo, failed to decode aggregated cosmetic entitlements", "error", err)
		return nil, errors.ErrInternalServerError()
	}

	// Map cosmetics
	cosmetics := []*structures.Cosmetic[bson.Raw]{}
	cur, err = r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameCosmetics).Find(
		ctx,
		bson.M{},
		options.Find().SetSort(bson.M{"priority": -1}),
	)

	if err != nil {
		ctx.Log().Errorw("mongo, failed to fetch cosmetics data", "error", err)
		return nil, errors.ErrInternalServerError()
	}

	if err = cur.All(ctx, &cosmetics); err != nil {
		ctx.Log().Errorw("mongo, failed to decode cosmetics data", "error", err)
		return nil, errors.ErrInternalServerError()
	}

	cosMap := make(map[primitive.ObjectID]*structures.Cosmetic[bson.Raw])

	for _, cos := range cosmetics {
		cosMap[cos.ID] = cos
	}

	// Structure entitlements by kind
	// kind:ent_id:[]ent
	ents := make(map[structures.EntitlementKind][]structures.Entitlement[bson.Raw])
	for _, ent := range entitlements {
		a := ents[ent.Kind]
		if a == nil {
			ents[ent.Kind] = []structures.Entitlement[bson.Raw]{ent}
		} else {
			ents[ent.Kind] = append(a, ent)
		}
	}

	// Map user IDs by roles
	roleMap := make(map[primitive.ObjectID][]primitive.ObjectID)

	for _, ent := range ents[structures.EntitlementKindRole] {
		r, err := structures.ConvertEntitlement[structures.EntitlementDataRole](ent)
		if err != nil {
			ctx.Log().Errorw("cosmetics, failed to convert entitlement", "error", err)
			return nil, errors.ErrInternalServerError()
		}

		if a := roleMap[ent.UserID]; a != nil {
			roleMap[ent.UserID] = append(roleMap[ent.UserID], r.Data.RefID)
		} else {
			roleMap[ent.UserID] = []primitive.ObjectID{r.Data.RefID}
		}
	}

	// Check entitled paints / badges for users we need to fetch
	entitledUserCount := 0
	entitledUserIDs := make([]primitive.ObjectID, len(ents[structures.EntitlementKindBadge])+len(ents[structures.EntitlementKindPaint]))
	userCosmetics := make(map[primitive.ObjectID][2]primitive.ObjectID) // [0] has badge, [1] has paint

	for _, ent := range ents[structures.EntitlementKindBadge] {
		if ok, d := readEntitled(roleMap[ent.UserID], ent); ok {
			uc := userCosmetics[ent.UserID]
			cos := cosMap[d.RefID]

			if !uc[0].IsZero() {
				oldCos := cosMap[uc[0]]
				if oldCos.ID.IsZero() || oldCos.Priority >= cos.Priority {
					continue // skip if priority is lower
				}
				// Find index of old
				for i, id := range oldCos.UserIDs {
					if id == ent.UserID {
						oldCos.UserIDs[i] = oldCos.UserIDs[len(oldCos.UserIDs)-1]
						oldCos.UserIDs = oldCos.UserIDs[:len(oldCos.UserIDs)-1]

						break
					}
				}
			}

			uc[0] = cos.ID
			cos.UserIDs = append(cos.UserIDs, ent.UserID)

			userCosmetics[ent.UserID] = uc
			entitledUserIDs[entitledUserCount] = ent.UserID
			entitledUserCount++
		}
	}

	for _, ent := range ents[structures.EntitlementKindPaint] {
		if ok, d := readEntitled(roleMap[ent.UserID], ent); ok {
			uc := userCosmetics[ent.UserID]
			if uc[1].IsZero() {
				cos := cosMap[d.RefID]
				cos.UserIDs = append(cos.UserIDs, ent.UserID)
				uc[1] = cos.ID
			}

			userCosmetics[ent.UserID] = uc
			entitledUserIDs[entitledUserCount] = ent.UserID
			entitledUserCount++
		}
	}
	// At this point we can fetch our users
	cur, err = r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).Find(ctx, bson.M{
		"_id": bson.M{"$in": entitledUserIDs[:entitledUserCount]},
	}, options.Find().SetProjection(bson.M{
		"_id":                    1,
		"connections.id":         1,
		"connections.platform":   1,
		"connections.data.login": 1,
	}))
	if err != nil {
		ctx.Log().Errorw("mongo, failed to spawn cosmetic users cursor", "error", err)
		return nil, errors.ErrInternalServerError()
	}

	// Decode data
	users := []structures.User{}
	if err = cur.All(ctx, &users); err != nil {
		ctx.Log().Errorw("mongo, failed to decode cosmetic users", "error", err)
		return nil, errors.ErrInternalServerError()
	}

	userMap := make(map[primitive.ObjectID]structures.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	// Find directly assigned users
	result := model.CosmeticsMap{
		Timestamp: time.Now().UnixMilli(),
		Badges:    []*model.CosmeticBadge{},
		Paints:    []*model.CosmeticPaint{},
	}

	for _, cos := range cosmetics {
		if len(cos.UserIDs) == 0 {
			continue // skip if cosmetic has no users
		}

		cos.Users = make([]structures.User, len(cos.UserIDs))

		for i, uid := range cos.UserIDs {
			cos.Users[i] = userMap[uid]
		}

		switch cos.Kind {
		case structures.CosmeticKindBadge:
			badge, _ := structures.ConvertCosmetic[structures.CosmeticDataBadge](*cos)
			urls := make([][2]string, 3)

			for i := 1; i <= 3; i++ {
				a := [2]string{}
				a[0] = strconv.Itoa(i)
				a[1] = fmt.Sprintf("https://%s/badge/%s/%dx", r.Ctx.Config().CdnURL, badge.ID.Hex(), i)
				urls[i-1] = a
			}

			b := createBadgeResponse(r.Ctx, badge.ToRaw(), cos.Users, idType)
			result.Badges = append(result.Badges, &b)
		case structures.CosmeticKindNametagPaint:
			paint, _ := structures.ConvertCosmetic[structures.CosmeticDataPaint](*cos)
			stops := make([]structures.CosmeticPaintGradientStop, len(paint.Data.Stops))
			dropShadows := make([]structures.CosmeticPaintDropShadow, len(paint.Data.DropShadows))

			for i, stop := range paint.Data.Stops {
				stops[i] = structures.CosmeticPaintGradientStop{
					At:    stop.At,
					Color: stop.Color,
				}
			}

			for i, shadow := range paint.Data.DropShadows {
				dropShadows[i] = structures.CosmeticPaintDropShadow{
					OffsetX: shadow.OffsetX,
					OffsetY: shadow.OffsetY,
					Radius:  shadow.Radius,
					Color:   shadow.Color,
				}
			}

			f := strings.Replace(string(paint.Data.Function), "_", "-", 1)
			f = strings.ToLower(f)
			paint.Data.Function = structures.CosmeticPaintFunction(f)

			b := createPaintResponse(paint.ToRaw(), cos.Users, idType)
			result.Paints = append(result.Paints, &b)
		}
	}

	return &result, nil
}

func readEntitled(roleList []primitive.ObjectID, ent structures.Entitlement[bson.Raw]) (bool, structures.EntitlementDataBaseSelectable) {
	d, _ := structures.ConvertEntitlement[structures.EntitlementDataBaseSelectable](ent)

	if !d.Data.Selected {
		return false, d.Data
	}

	if len(d.Condition.AllRoles) > 0 {
		for _, rol := range d.Condition.AllRoles {
			if !utils.Contains(roleList, rol) {
				return false, d.Data
			}
		}
	}

	if len(d.Condition.AnyRoles) > 0 {
		for _, rol := range d.Condition.AnyRoles {
			if utils.Contains(roleList, rol) {
				continue
			}
		}
	}

	return true, d.Data
}

func createBadgeResponse(gctx global.Context, badge structures.Cosmetic[bson.Raw], users []structures.User, idType string) model.CosmeticBadge {
	// Get user list
	userIDs := selectUserIDType(users, idType)

	// Generate URLs
	urls := make([][2]string, 3)

	for i := 1; i <= 3; i++ {
		a := [2]string{}
		a[0] = fmt.Sprintf("%d", i)
		a[1] = fmt.Sprintf("https://%s/badge/%s/%dx", gctx.Config().CdnURL, badge.ID.Hex(), i)

		urls[i-1] = a
	}

	data, _ := structures.ConvertCosmetic[structures.CosmeticDataBadge](badge)

	response := model.CosmeticBadge{
		ID:      badge.ID.Hex(),
		Name:    badge.Name,
		Tooltip: data.Data.Tooltip,
		Users:   userIDs,
		URLs:    urls,
		Misc:    data.Data.Misc,
	}

	return response
}

func createPaintResponse(paint structures.Cosmetic[bson.Raw], users []structures.User, idType string) model.CosmeticPaint {
	// Get user list
	userIDs := selectUserIDType(users, idType)

	data, _ := structures.ConvertCosmetic[structures.CosmeticDataPaint](paint)

	stops := make([]model.CosmeticPaintGradientStop, len(data.Data.Stops))
	for i, stop := range data.Data.Stops {
		stops[i] = model.CosmeticPaintGradientStop{
			At:    stop.At,
			Color: stop.Color.Sum(),
		}
	}

	shadows := make([]model.CosmeticPaintDropShadow, len(data.Data.DropShadows))
	for i, shadow := range data.Data.DropShadows {
		shadows[i] = model.CosmeticPaintDropShadow{
			OffsetX: shadow.OffsetX,
			OffsetY: shadow.OffsetY,
			Radius:  shadow.Radius,
			Color:   shadow.Color.Sum(),
		}
	}

	var color *int32
	if data.Data.Color != nil {
		color = utils.PointerOf(data.Data.Color.Sum())
	}

	return model.CosmeticPaint{
		ID:          paint.ID.Hex(),
		Name:        paint.Name,
		Users:       userIDs,
		Color:       color,
		Function:    string(data.Data.Function),
		Stops:       stops,
		Repeat:      data.Data.Repeat,
		Angle:       data.Data.Angle,
		Shape:       data.Data.Shape,
		ImageURL:    data.Data.ImageURL,
		DropShadows: shadows,
	}
}

func selectUserIDType(users []structures.User, t string) []string {
	userIDs := make([]string, len(users))

	for i, u := range users {
		if u.ID.IsZero() {
			continue
		}

		switch t {
		case "object_id":
			userIDs[i] = u.ID.Hex()
		case "twitch_id":
			tw, _, _ := u.Connections.Twitch()
			if tw.ID == "" {
				continue
			}

			userIDs[i] = tw.ID
		case "login":
			tw, _, _ := u.Connections.Twitch()
			if tw.Data.Login == "" {
				continue
			}

			userIDs[i] = tw.Data.Login
		}
	}

	return userIDs
}
