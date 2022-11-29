package query

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/aggregations"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func (q *Query) EmoteSets(ctx context.Context, filter bson.M, opts ...QueryEmoteSetsOptions) *QueryResult[structures.EmoteSet] {
	qr := &QueryResult[structures.EmoteSet]{}
	items := []structures.EmoteSet{}

	opt := QueryEmoteSetsOptions{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	// Fetch Emote Sets
	cur, err := q.mongo.Collection(mongo.CollectionNameEmoteSets).Aggregate(ctx, aggregations.Combine(
		mongo.Pipeline{{{Key: "$match", Value: filter}}},
		utils.Ternary(opt.FetchOrigins, mongo.Pipeline{
			{{
				Key: "$lookup",
				Value: mongo.Lookup{
					From:         mongo.CollectionNameEmoteSets,
					LocalField:   "origins.id",
					ForeignField: "_id",
					As:           "origin_sets",
				},
			}},
		}, mongo.Pipeline{}),
		mongo.Pipeline{{{
			Key: "$project",
			Value: bson.M{
				"set": "$$ROOT",
				"origin_sets": bson.M{
					"$arrayToObject": bson.A{
						bson.M{"$map": bson.M{
							"input": "$origin_sets",
							"in": bson.M{
								"k": bson.M{"$toString": "$$this._id"},
								"v": "$$this",
							},
						}},
					},
				}},
		}}},
	))
	if err != nil {
		zap.S().Errorw("mongo, failed to query emote sets", "error", err)

		return qr.setError(errors.ErrInternalServerError())
	}

	sets := []aggregatedSetWithOrigins{}
	if err = cur.All(ctx, &sets); err != nil {
		zap.S().Errorw("mongo, failed to fetch emote sets", "error", err)

		return qr.setError(errors.ErrInternalServerError())
	}

	// Get IDs of relational data
	userIDs := make(utils.Set[primitive.ObjectID])
	emoteIDs := make(utils.Set[primitive.ObjectID])

	for _, set := range sets {
		userIDs.Add(set.Set.OwnerID)

		for _, emote := range set.Set.Emotes {
			userIDs.Add(emote.ActorID)
			emoteIDs.Add(emote.ID)
		}
	}

	for _, set := range sets {
		emoteMap := make(map[primitive.ObjectID]int)
		emoteNameMap := make(map[string]int)

		emotes := make([]structures.ActiveEmote, len(set.Set.Emotes))
		copy(emotes, set.Set.Emotes)

		for i, ae := range emotes {
			emoteMap[ae.ID] = i
			emoteNameMap[ae.Name] = i
		}

		// Apply emotes from origins
		for _, origin := range set.Set.Origins {
			subset := set.OriginSets[origin.ID]
			if subset.ID.IsZero() {
				continue // set wasn't found
			}

			for _, ae := range subset.Emotes {
				if len(emotes) >= int(set.Set.Capacity) {
					break
				}

				ae.Origin = origin

				if ix, ok := emoteMap[ae.ID]; ok {
					emotes[ix] = ae
				} else if ix, ok := emoteNameMap[ae.Name]; ok {
					emotes[ix] = ae
				} else {
					emotes = append(emotes, ae)
				}
			}

			// resize emotes slice
			set.Set.Emotes = emotes
		}

		items = append(items, set.Set)
	}

	return qr.setItems(items)
}

type aggregatedSetWithOrigins struct {
	Set        structures.EmoteSet                        `bson:"set"`
	OriginSets map[primitive.ObjectID]structures.EmoteSet `bson:"origin_sets"`
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
	if err != nil {
		return nil, err
	}

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
