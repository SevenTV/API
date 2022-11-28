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
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func (q *Query) EmoteSets(ctx context.Context, filter bson.M) *QueryResult[structures.EmoteSet] {
	qr := &QueryResult[structures.EmoteSet]{}
	items := []structures.EmoteSet{}

	// Fetch Emote Sets
	cur, err := q.mongo.Collection(mongo.CollectionNameEmoteSets).Find(ctx, filter)
	if err != nil {
		zap.S().Errorw("mongo, failed to query emote sets", "error", err)

		return qr.setError(errors.ErrInternalServerError())
	}

	sets := []structures.EmoteSet{}
	if err = cur.All(ctx, &sets); err != nil {
		zap.S().Errorw("mongo, failed to fetch emote sets", "error", err)

		return qr.setError(errors.ErrInternalServerError())
	}

	// Get IDs of relational data
	userIDs := make(utils.Set[primitive.ObjectID])
	emoteIDs := make(utils.Set[primitive.ObjectID])

	for _, set := range sets {
		userIDs.Add(set.OwnerID)

		for _, emote := range set.Emotes {
			userIDs.Add(emote.ActorID)
			emoteIDs.Add(emote.ID)
		}
	}

	// Fetch emotes
	cur, err = q.mongo.Collection(mongo.CollectionNameEmotes).Find(ctx, bson.M{
		"versions.id": bson.M{"$in": emoteIDs.Values()},
	}, options.Find().SetProjection(bson.M{
		"owner_id":                          1,
		"name":                              1,
		"flags":                             1,
		"versions.id":                       1,
		"versions.state":                    1,
		"versions.animated":                 1,
		"versions.image_files.name":         1,
		"versions.image_files.width":        1,
		"versions.image_files.height":       1,
		"versions.image_files.size":         1,
		"versions.image_files.key":          1,
		"versions.image_files.content_type": 1,
	}))
	if err != nil {
		zap.S().Errorw("mongo, failed to query relational emotes in emote sets", "error", err)

		return qr.setError(errors.ErrInternalServerError())
	}

	emotes := []structures.Emote{}
	if err = cur.All(ctx, &emotes); err != nil {
		zap.S().Errorw("mongo, failed to fetch relational emotes of emote sets", "error", err)

		return qr.setError(errors.ErrInternalServerError())
	}

	for _, e := range emotes {
		userIDs.Add(e.OwnerID)
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

	emoteMap := make(map[primitive.ObjectID]structures.Emote)
	for _, emote := range emotes {
		owner := userMap[emote.OwnerID]
		if !owner.ID.IsZero() {
			emote.Owner = &owner
		}

		for _, ver := range emote.Versions {
			emote.ID = ver.ID
			emoteMap[ver.ID] = emote
		}
	}

	for _, set := range sets {
		owner := userMap[set.OwnerID]
		if !owner.ID.IsZero() {
			set.Owner = &owner
		}
		for indEmotes, ae := range set.Emotes {
			emote, ok := emoteMap[ae.ID]

			if !ok {
				set.Emotes[indEmotes].Emote = &structures.DeletedEmote
			} else {
				set.Emotes[indEmotes].Emote = &emote
			}

			// Apply actor user to active emote data?
			if ae.ActorID.IsZero() {
				continue
			}

			if actor, ok := userMap[ae.ActorID]; ok {
				set.Emotes[indEmotes].Actor = &actor
			}
		}

		items = append(items, set)
	}

	return qr.setItems(items)
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
						"$push": "$$ROOT",
					},
				},
			}},
			{{
				Key: "$lookup",
				Value: mongo.Lookup{
					From:         mongo.CollectionNameEmotes,
					LocalField:   "sets.emotes.id",
					ForeignField: "versions.id",
					As:           "emotes",
				},
			}},
			{{
				Key: "$set",
				Value: bson.M{
					"all_users": bson.M{
						"$setUnion": bson.A{"$sets.owner_id", "$sets.emotes.owner_id"},
					},
				},
			}},
			{{
				Key: "$lookup",
				Value: mongo.Lookup{
					From:         mongo.CollectionNameUsers,
					LocalField:   "all_users",
					ForeignField: "_id",
					As:           "users",
				},
			}},
			{{
				Key: "$lookup",
				Value: mongo.Lookup{
					From:         mongo.CollectionNameEntitlements,
					LocalField:   "all_users",
					ForeignField: "user_id",
					As:           "role_entitlements",
				},
			}},
			{{
				Key: "$set",
				Value: bson.M{
					"role_entitlements": bson.M{
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
		},
	))
	if err != nil {
		return nil, err
	}

	// Iterate over cursor
	bans, err := q.Bans(ctx, BanQueryOptions{ // remove emotes made by usersa who own nothing and are happy
		Filter: bson.M{"effects": bson.M{"$bitsAnySet": structures.BanEffectNoOwnership | structures.BanEffectMemoryHole}},
	})
	if err != nil {
		return nil, err
	}

	for i := 0; cur.Next(ctx); i++ {
		v := &aggregatedUserEmoteSets{}
		if err = cur.Decode(v); err != nil {
			continue
		}

		// Map emotes bound to the set
		qb := &QueryBinder{ctx, q}
		userMap, err := qb.MapUsers(v.Users, v.RoleEntitlements...)
		if err != nil {
			return nil, err
		}

		emoteMap := make(map[primitive.ObjectID]structures.Emote)
		for _, emote := range v.Emotes {
			if _, ok := bans.NoOwnership[emote.OwnerID]; ok {
				continue
			}
			if _, ok := bans.MemoryHole[emote.OwnerID]; ok {
				emote.OwnerID = primitive.NilObjectID
			}
			for _, ver := range emote.Versions {
				emote.ID = ver.ID

				owner := userMap[emote.OwnerID]
				if !owner.ID.IsZero() {
					emote.Owner = &owner
				}

				emoteMap[ver.ID] = emote
			}
		}

		for idx, set := range v.Sets {
			for idx, ae := range set.Emotes {
				if emote, ok := emoteMap[ae.ID]; ok {
					emote.ID = ae.ID
					ae.Emote = &emote
					set.Emotes[idx] = ae
				}

				// Apply actor user to active emote data?
				if ae.ActorID.IsZero() {
					continue
				}

				if actor, ok := userMap[ae.ActorID]; ok {
					set.Emotes[idx].Actor = &actor
				}
			}
			v.Sets[idx] = set
		}
		items[v.UserID] = v.Sets
	}
	return items, multierror.Append(err, cur.Close(ctx)).ErrorOrNil()
}

type aggregatedUserEmoteSets struct {
	UserID           primitive.ObjectID                 `bson:"_id"`
	Sets             []structures.EmoteSet              `bson:"sets"`
	Emotes           []structures.Emote                 `bson:"emotes"`
	Users            []structures.User                  `bson:"users"`
	RoleEntitlements []structures.Entitlement[bson.Raw] `bson:"role_entitlements"`
}
