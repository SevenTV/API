package emote

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/api/internal/gql/v3/types"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type Resolver struct {
	types.Resolver

	mx *sync.Mutex
}

func New(r types.Resolver) generated.EmoteResolver {
	return &Resolver{r, &sync.Mutex{}}
}

func (r *Resolver) Images(ctx context.Context, obj *model.Emote, format []model.ImageFormat) ([]*model.Image, error) {
	return helpers.FilterImages(obj.Images, format), nil
}

func (r *Resolver) Owner(ctx context.Context, obj *model.Emote) (*model.User, error) {
	if obj.Owner != nil && obj.Owner.ID != structures.DeletedUser.ID {
		return obj.Owner, nil
	}

	user, err := r.Ctx.Inst().Loaders.UserByID().Load(obj.OwnerID)
	if err != nil && !errors.Compare(err, errors.ErrUnknownUser()) {
		return nil, err
	}

	return helpers.UserStructureToModel(user, r.Ctx.Config().CdnURL), nil
}

func (r *Resolver) Reports(ctx context.Context, obj *model.Emote) ([]*model.Report, error) {
	// TODO
	return nil, nil
}

func (r *Resolver) CommonNames(ctx context.Context, obj *model.Emote) ([]*model.EmoteCommonName, error) {
	r.mx.Lock()
	defer r.mx.Unlock()

	// Compose cache key
	cacheKey := r.Ctx.Inst().Redis.ComposeKey("rest", fmt.Sprintf("cache:emote:%s:common_names", obj.ID.Hex()))

	// Return existing cache?
	d, err := r.Ctx.Inst().Redis.Get(ctx, cacheKey)
	if err == nil && d != "" {
		result := []*model.EmoteCommonName{}
		if err = json.Unmarshal(utils.S2B(d), &result); err != nil {
			zap.S().Errorw("gql, failed to return cache of Emote/common_names")
		}

		return result, nil
	}

	// Query emote sets with the emote enabled, but only those with a different name
	// then project only the active emote object for this emote
	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmoteSets).Find(ctx, bson.M{"emotes": bson.M{
		"$elemMatch": bson.M{
			"id":   obj.ID,
			"name": bson.M{"$not": bson.M{"$eq": obj.Name}},
		},
	}}, options.Find().SetProjection(bson.M{"emotes": bson.M{
		"$elemMatch": bson.M{"id": obj.ID},
	}}))
	if err != nil {
		zap.S().Errorw("mongo, failed to spawn aggregation for emote common names", "error", err)
		return nil, errors.ErrInternalServerError()
	}

	// Fetch the data
	items := []emoteCommonName{}
	if err = cur.All(ctx, &items); err != nil {
		zap.S().Errorw("mongo, failed to retrieve common name variants of emote",
			"error", err,
			"emote_id", obj.ID,
		)

		return nil, errors.ErrInternalServerError()
	}

	// Build the result
	m := make(map[string]int)
	result := []*model.EmoteCommonName{}

	for _, n := range items {
		ae := n.Emotes[0]
		if ae.ID.IsZero() {
			continue
		}

		m[ae.Name]++
	}

	for n, c := range m {
		result = append(result, &model.EmoteCommonName{
			Name:  n,
			Count: c,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		a := result[i]
		b := result[j]
		return a.Count > b.Count
	})

	b, _ := json.Marshal(result)
	if err := r.Ctx.Inst().Redis.SetEX(ctx, cacheKey, utils.B2S(b), 8*time.Hour); err != nil {
		zap.S().Errorw("gql, couldn't save response of Emote/common_names to redis cache")
	}

	return result, nil
}

type emoteCommonName struct {
	Emotes [1]structures.ActiveEmote `json:"-" bson:"emotes"`
}
