package cosmetics

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/api/internal/rest/v2/model"
	"github.com/seventv/api/internal/rest/v3/middleware"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
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
			middleware.SetCacheControl(r.Ctx, 600, []string{"s-maxage=300"}),
		},
	}
}

// Get Cosmetics
// @Summary Get Cosmetics
// @Description Get all active cosmetics and the users assigned to them
// @Tags cosmetics
// @Param user_identifier query string false "one of 'object_id', 'twitch_id' or 'login'"
// @Produce json
// @Success 200 {object} model.CosmeticsMap
// @Router /cosmetics [get]
func (r *Route) Handler(ctx *rest.Ctx) errors.APIError {
	r.mx.Lock()
	defer r.mx.Unlock()

	// identifier type argument
	idType := utils.B2S(ctx.QueryArgs().Peek("user_identifier"))

	if !utils.Contains([]string{"object_id", "twitch_id", "login"}, idType) {
		return errors.ErrInvalidRequest().SetDetail("Query Param 'user_identifier' must be 'object_id', 'twitch_id' or 'login'")
	}

	// Compose cache key
	cacheKey := r.Ctx.Inst().Redis.ComposeKey("rest", fmt.Sprintf("cache:cosmetics:%s", idType))

	// Return existing cache?
	d, err := r.Ctx.Inst().Redis.Get(ctx, cacheKey)
	if err == nil && d != "" {
		result := &model.CosmeticsMap{}
		if err := json.Unmarshal(utils.S2B(d), result); err != nil {
			zap.S().Errorw("failed to return cache of /v2/cosmetics",
				"error", err,
			)
		}

		return ctx.JSON(rest.OK, result)
	}

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
	})
	if err != nil {
		zap.S().Errorw("mongo, failed to spawn cosmetic entitlements aggregation", "error", err)
		return errors.ErrInternalServerError()
	}

	// Decode data
	entitlements := []structures.Entitlement[bson.Raw]{}
	if err = cur.All(ctx, &entitlements); err != nil {
		zap.S().Errorw("mongo, failed to decode aggregated cosmetic entitlements", "error", err)
		return errors.ErrInternalServerError()
	}

	// Map cosmetics
	cosmetics := []*structures.Cosmetic[bson.Raw]{}
	cur, err = r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameCosmetics).Find(
		ctx,
		bson.M{},
		options.Find().SetSort(bson.M{"priority": -1}),
	)

	if err != nil {
		zap.S().Errorw("mongo, failed to fetch cosmetics data", "error", err)
		return errors.ErrInternalServerError()
	}

	if err = cur.All(ctx, &cosmetics); err != nil {
		zap.S().Errorw("mongo, failed to decode cosmetics data", "error", err)
		return errors.ErrInternalServerError()
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
			zap.S().Errorw("cosmetics, failed to convert entitlement", "error", err)
			return errors.ErrInternalServerError()
		}

		if a := roleMap[ent.UserID]; a != nil {
			roleMap[ent.UserID] = append(roleMap[ent.UserID], r.Data.ObjectReference)
		} else {
			roleMap[ent.UserID] = []primitive.ObjectID{r.Data.ObjectReference}
		}
	}

	// Check entitled paints / badges for users we need to fetch
	entitledUserCount := 0
	entitledUserIDs := make([]primitive.ObjectID, len(ents[structures.EntitlementKindBadge])+len(ents[structures.EntitlementKindPaint]))
	userCosmetics := make(map[primitive.ObjectID][2]primitive.ObjectID) // [0] has badge, [1] has paint

	for _, ent := range ents[structures.EntitlementKindBadge] {
		if ok, d := readEntitled(roleMap[ent.UserID], ent); ok {
			uc := userCosmetics[ent.UserID]
			cos := cosMap[d.ObjectReference]

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
				cos := cosMap[d.ObjectReference]
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
		"_id":                  1,
		"connections.id":       1,
		"connections.platform": 1,
		"username":             1,
	}))
	if err != nil {
		zap.S().Errorw("mongo, failed to spawn cosmetic users cursor", "error", err)
		return errors.ErrInternalServerError()
	}

	// Decode data
	users := []structures.User{}
	if err = cur.All(ctx, &users); err != nil {
		zap.S().Errorw("mongo, failed to decode cosmetic users", "error", err)
		return errors.ErrInternalServerError()
	}

	userMap := make(map[primitive.ObjectID]structures.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	// Find directly assigned users
	result := GetCosmeticsResult{
		Badges: []badgeCosmeticResponse{},
		Paints: []paintCosmeticResponse{},
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

			b := createBadgeResponse(r.Ctx, *cos, cos.Users, idType)
			result.Badges = append(result.Badges, b)
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

			b := createPaintResponse(*cos, cos.Users, idType)
			result.Paints = append(result.Paints, b)
		}
	}

	b, _ := json.Marshal(result)
	if err := r.Ctx.Inst().Redis.SetEX(ctx, cacheKey, utils.B2S(b), 10*time.Minute); err != nil {
		logrus.WithField("id_type", idType).WithError(err).Error("couldn't save cosmetics response to redis cache")
	}

	return ctx.JSON(rest.OK, result)
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

type GetCosmeticsResult struct {
	Badges []badgeCosmeticResponse `json:"badges"`
	Paints []paintCosmeticResponse `json:"paints"`
}
type badgeCosmeticResponse struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`
	Tooltip string     `json:"tooltip"`
	URLs    [][]string `json:"urls"`
	Users   []string   `json:"users"`
	Misc    bool       `json:"misc,omitempty"`
}

type paintCosmeticResponse struct {
	ID          string                                 `json:"id"`
	Name        string                                 `json:"name"`
	Users       []string                               `json:"users"`
	Function    string                                 `json:"function"`
	Color       *int32                                 `json:"color"`
	Stops       []structures.CosmeticPaintGradientStop `json:"stops"`
	Repeat      bool                                   `json:"repeat"`
	Angle       int32                                  `json:"angle"`
	Shape       string                                 `json:"shape,omitempty"`
	ImageURL    string                                 `json:"image_url,omitempty"`
	DropShadow  structures.CosmeticPaintDropShadow     `json:"drop_shadow,omitempty"`
	DropShadows []structures.CosmeticPaintDropShadow   `json:"drop_shadows,omitempty"`
	Animation   structures.CosmeticPaintAnimation      `json:"animation,omitempty"`
}

func createBadgeResponse(gctx global.Context, badge structures.Cosmetic[bson.Raw], users []structures.User, idType string) badgeCosmeticResponse {
	// Get user list
	userIDs := selectUserIDType(users, idType)

	// Generate URLs
	urls := make([][]string, 3)

	for i := 1; i <= 3; i++ {
		a := make([]string, 2)
		a[0] = fmt.Sprintf("%d", i)
		a[1] = fmt.Sprintf("https://%s/badge/%s/%d", gctx.Config().CdnURL, badge.ID.Hex(), i)

		urls[i-1] = a
	}

	data, _ := structures.ConvertCosmetic[structures.CosmeticDataBadge](badge)

	response := badgeCosmeticResponse{
		ID:      badge.ID.Hex(),
		Name:    badge.Name,
		Tooltip: data.Data.Tooltip,
		Users:   userIDs,
		URLs:    urls,
		Misc:    data.Data.Misc,
	}

	return response
}

func createPaintResponse(paint structures.Cosmetic[bson.Raw], users []structures.User, idType string) paintCosmeticResponse {
	// Get user list
	userIDs := selectUserIDType(users, idType)

	data, _ := structures.ConvertCosmetic[structures.CosmeticDataPaint](paint)

	return paintCosmeticResponse{
		ID:          paint.ID.Hex(),
		Name:        paint.Name,
		Users:       userIDs,
		Color:       data.Data.Color,
		Function:    string(data.Data.Function),
		Stops:       data.Data.Stops,
		Repeat:      data.Data.Repeat,
		Angle:       data.Data.Angle,
		Shape:       data.Data.Shape,
		ImageURL:    data.Data.ImageURL,
		DropShadows: data.Data.DropShadows,
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
			userIDs[i] = u.Username
		}
	}

	return userIDs
}
