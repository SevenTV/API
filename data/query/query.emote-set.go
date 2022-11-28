package query

import (
	"context"
	"sort"

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

func (q *Query) EmoteSets(ctx context.Context, filter bson.M) *QueryResult[structures.EmoteSet] {
	qr := &QueryResult[structures.EmoteSet]{}
	items := []structures.EmoteSet{}

	// Fetch Emote Sets
	cur, err := q.mongo.Collection(mongo.CollectionNameEmoteSets).Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{
			Key: "$lookup",
			Value: mongo.Lookup{
				From:         mongo.CollectionNameEmoteSets,
				LocalField:   "origins.id",
				ForeignField: "_id",
				As:           "origin_sets",
			},
		}},
		{{
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
		}},
	})
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

	// Fetch users
	cur, err = q.mongo.Collection(mongo.CollectionNameUsers).Aggregate(ctx, mongo.Pipeline{
		{{
			Key: "$match",
			Value: bson.M{
				"_id": bson.M{"$in": userIDs.Values()},
			},
		}},
		{{
			Key: "$lookup",
			Value: mongo.Lookup{
				From:         mongo.CollectionNameEntitlements,
				LocalField:   "_id",
				ForeignField: "user_id",
				As:           "role_entitlements",
			},
		}},
		{{
			Key: "$set",
			Value: bson.M{
				"entitlements": bson.M{
					"$filter": bson.M{
						"input": "$role_entitlements",
						"as":    "ent",
						"cond": bson.M{
							"$eq": bson.A{"$$ent.kind", structures.EntitlementKindRole},
						},
					},
				},
			},
		}},
	})
	if err != nil {
		zap.S().Errorw("mongo, failed to query relational users of emote sets", "error", err)

		return qr.setError(errors.ErrInternalServerError())
	}

	users := []structures.User{}
	if err = cur.All(ctx, &users); err != nil {
		zap.S().Errorw("mongo, failed to fetch relational users of emote sets", "error", err)

		return qr.setError(errors.ErrInternalServerError())
	}

	qb := &QueryBinder{ctx, q}

	userMap, err := qb.MapUsers(users)
	if err != nil {
		return qr.setError(err)
	}

	for _, set := range sets {
		owner := userMap[set.Set.OwnerID]
		if !owner.ID.IsZero() {
			set.Set.Owner = &owner
		}

		emoteMap := make(map[string]structures.ActiveEmote)
		for _, ae := range set.Set.Emotes {
			emoteMap[ae.Name] = ae
		}

		// Apply emotes from origins
		for _, origin := range set.Set.Origins {
			subset := set.OriginSets[origin.ID]
			if subset.ID.IsZero() {
				continue // set wasn't found
			}

			startAt := len(emoteMap)

			// resize emotes slice
			for pos, ae := range subset.Emotes {
				i := startAt + pos

				// overcapacity:
				// reduce slice size and add no more emotes
				if i > int(set.Set.Capacity) {
					break
				}

				// test weight
				if x, ok := emoteMap[ae.Name]; ok && (!x.Origin.ID.IsZero() && x.Origin.Weight >= ae.Origin.Weight || origin.Weight < 0) {
					continue
				}

				ae.Origin = origin
				emoteMap[ae.Name] = ae
			}
		}

		set.Set.Emotes = make([]structures.ActiveEmote, len(emoteMap))

		i := -1
		for _, emote := range emoteMap {
			i++

			set.Set.Emotes[i] = emote
		}

		sort.Slice(set.Set.Emotes, func(i, j int) bool {
			return set.Set.Emotes[i].Timestamp.After(set.Set.Emotes[j].Timestamp)
		})

		items = append(items, set.Set)
	}

	return qr.setItems(items)
}

type aggregatedSetWithOrigins struct {
	Set        structures.EmoteSet                        `bson:"set"`
	OriginSets map[primitive.ObjectID]structures.EmoteSet `bson:"origin_sets"`
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
		}).Items()
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
