package emote

import (
	"context"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/api/internal/gql/v3/types"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	// Disable for now because this is too heavy on the DB
	return nil, nil
}

/* TODO: Find a more optimized way to do this
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

i := 0

for n, c := range m {
	result = append(result, &model.EmoteCommonName{
		Name:  n,
		Count: c,
	})

	i++

	if i+1 > 10 {
		break
	}
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

return result, nil */

/*
type emoteCommonName struct {
	Emotes [1]structures.ActiveEmote `json:"-" bson:"emotes"`
}
*/

func (r *Resolver) Activity(ctx context.Context, obj *model.Emote, limitArg *int) ([]*model.AuditLog, error) {
	result := []*model.AuditLog{}

	limit := 50
	if limitArg != nil {
		limit = *limitArg

		if limit > 300 {
			return result, errors.ErrInvalidRequest().SetDetail("limit must be less than 300")
		} else if limit < 1 {
			return result, errors.ErrInvalidRequest().SetDetail("limit must be greater than 0")
		}
	}

	logs := []structures.AuditLog{}
	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameAuditLogs).Find(ctx, bson.M{
		"kind":        bson.M{"$gte": 1, "$lte": 19},
		"target_id":   obj.ID,
		"target_kind": structures.ObjectKindEmote,
	}, options.Find().SetSort(bson.M{"_id": -1}).SetLimit(int64(limit)))

	if err != nil {
		return result, errors.ErrInternalServerError()
	}

	if err := cur.All(ctx, &logs); err != nil {
		return result, errors.ErrInternalServerError()
	}

	actorMap := make(map[primitive.ObjectID]structures.User)

	for _, l := range logs {
		a := model.AuditLog{
			ID:         l.ID,
			Kind:       int(l.Kind),
			ActorID:    l.ActorID,
			TargetID:   l.TargetID,
			TargetKind: int(l.TargetKind),
			CreatedAt:  l.ID.Timestamp(),
			Changes:    make([]*model.AuditLogChange, len(l.Changes)),
			Reason:     l.Reason,
		}

		actorMap[l.ActorID] = structures.DeletedUser

		// Append changes
		for i, c := range l.Changes {
			val := map[string]any{}
			aryval := model.AuditLogChangeArray{}

			switch c.Format {
			case structures.AuditLogChangeFormatSingleValue:
				_ = bson.Unmarshal(c.Value, &val)
			case structures.AuditLogChangeFormatArrayChange:
				_ = bson.Unmarshal(c.Value, &aryval)
			}

			a.Changes[i] = &model.AuditLogChange{
				Format:     int(c.Format),
				Key:        c.Key,
				Value:      val,
				ArrayValue: &aryval,
			}
		}

		result = append(result, &a)
	}

	// Fetch and add actors to the result

	i := 0
	actorIDs := make([]primitive.ObjectID, len(actorMap))

	for oid := range actorMap {
		actorIDs[i] = oid
		i++
	}

	actors, errs := r.Ctx.Inst().Loaders.UserByID().LoadAll(actorIDs)
	if multierror.Append(nil, errs...).ErrorOrNil() != nil {
		return result, errors.ErrInternalServerError()
	}

	for _, u := range actors {
		actorMap[u.ID] = u
	}

	// Add actors to result
	for i, l := range result {
		result[i].Actor = helpers.UserStructureToPartialModel(helpers.UserStructureToModel(actorMap[l.ActorID], r.Ctx.Config().CdnURL))
	}

	return result, nil
}
