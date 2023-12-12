package query

import (
	"context"
	"sort"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/aggregations"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func (q *Query) EmoteSets(ctx context.Context, filter bson.M, opts ...QueryEmoteSetsOptions) *QueryResult[structures.EmoteSet] {
	qr := &QueryResult[structures.EmoteSet]{}
	items := []structures.EmoteSet{}

	cur, err := q.mongo.Collection(mongo.CollectionNameEmoteSets).Find(
		ctx,
		filter,
		options.Find().SetBatchSize(10).SetNoCursorTimeout(true),
	)
	if err != nil {
		zap.S().Errorw("mongo, failed to query emote sets", "error", err)
		return qr.setError(err)
	}

	defer cur.Close(ctx)

	if err = cur.All(ctx, &items); err != nil {
		zap.S().Errorw("mongo, failed to fetch emote sets", "error", err)

		return qr.setError(errors.ErrInternalServerError())
	}

	shouldGetOrigin := false

	for _, opt := range opts {
		if opt.FetchOrigins {
			shouldGetOrigin = true
			break
		}
	}

	if !shouldGetOrigin {
		return qr.setItems(items)
	}

	originsToFetch := getOriginIds(items)
	// if there are no origins to fetch, return
	if len(originsToFetch) == 0 {
		return qr.setItems(items)
	}

	// fetch origins
	originSets, err := q.EmoteSets(ctx, bson.M{"_id": bson.M{"$in": originsToFetch}}).Items()
	if err != nil {
		zap.S().Errorw("mongo, failed to fetch origin sets", "error", err)
		return qr.setError(err)
	}

	return qr.setItems(applyOriginSets(items, originSets))
}

// applyOriginSets inserts origin sets into the emote sets
func applyOriginSets(sets []structures.EmoteSet, originSets []structures.EmoteSet) []structures.EmoteSet {
	originMap := make(map[primitive.ObjectID]structures.EmoteSet)
	for _, set := range originSets {
		originMap[set.ID] = set
	}

	for i, set := range sets {
		emoteNameIndexes := make(map[string]int)
		for emoteIndex, emote := range set.Emotes {
			emoteNameIndexes[emote.Name] = emoteIndex
		}

		for originIndex, origin := range set.Origins {
			subset := originMap[origin.ID]
			if subset.ID.IsZero() {
				continue // set wasn't found
			}

			origin.Set = &subset

			for _, emote := range subset.Emotes {
				if index, ok := emoteNameIndexes[emote.Name]; ok {
					if set.Emotes[index].ID == emote.ID {
						continue
					}

					emote.Origin = origin
					set.Emotes[index] = emote
				} else {
					// don't exceed capacity, but we still replace other origin set emotes
					if len(set.Emotes) >= int(set.Capacity) {
						continue
					}

					// add emote to set emotes
					emoteNameIndexes[emote.Name] = len(set.Emotes)
					set.Emotes = append(set.Emotes, emote)
				}
			}

			// TODO: figure out of this sort is needed
			sort.Slice(set.Emotes, func(i, j int) bool {
				return set.Emotes[i].Origin.ID.IsZero()
			})

			set.Origins[originIndex] = origin
		}

		sets[i] = set
	}

	return sets
}

func getOriginIds(items []structures.EmoteSet) []primitive.ObjectID {
	ids := make([]primitive.ObjectID, 0)

	for _, set := range items {
		for _, origin := range set.Origins {
			// make sure we only add unique ids
			matched := false

			for _, id := range ids {
				if id == origin.ID {
					matched = true
					break
				}
			}

			if !matched {
				ids = append(ids, origin.ID)
			}
		}
	}

	return ids
}

type QueryEmoteSetsOptions struct {
	FetchOrigins bool
}

func (q *Query) UserEmoteSets(ctx context.Context, filter bson.M) (map[primitive.ObjectID][]structures.EmoteSet, error) {
	items := make(map[primitive.ObjectID][]structures.EmoteSet)

	cur, err := q.mongo.Collection(mongo.CollectionNameEmoteSets).Aggregate(ctx, aggregations.Combine(
		mongo.Pipeline{
			{{
				Key:   "$match",
				Value: filter,
			}},
			{{
				Key: "$group",
				Value: bson.M{
					"_id": "$owner_id",
					"sets": bson.M{
						"$push": "$$ROOT._id",
					},
				},
			}},
		},
	))
	if err != nil {
		return nil, err
	}

	// Iterate over cursor
	for i := 0; cur.Next(ctx); i++ {
		v := &aggregatedUserEmoteSets{}
		if err = cur.Decode(v); err != nil {
			continue
		}

		sets, err := q.EmoteSets(ctx, bson.M{
			"_id": bson.M{"$in": v.Sets},
		}, QueryEmoteSetsOptions{FetchOrigins: true}).Items()
		if err != nil {
			continue
		}

		items[v.UserID] = sets
	}

	return items, multierror.Append(err, cur.Close(ctx)).ErrorOrNil()
}

type aggregatedUserEmoteSets struct {
	UserID primitive.ObjectID   `bson:"_id"`
	Sets   []primitive.ObjectID `bson:"sets"`
}
