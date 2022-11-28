package query

import (
	"context"
	"io"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (q *Query) Emotes(ctx context.Context, filter bson.M) *QueryResult[structures.Emote] {
	qr := QueryResult[structures.Emote]{}
	items := []structures.Emote{}

	bans, err := q.Bans(ctx, BanQueryOptions{
		Filter: bson.M{"effects": bson.M{"$bitsAnySet": structures.BanEffectNoOwnership | structures.BanEffectMemoryHole}},
	})
	if err != nil {
		return qr.setError(err)
	}

	cur, err := q.mongo.Collection(mongo.CollectionNameEmotes).Aggregate(ctx, mongo.Pipeline{
		{{
			Key:   "$match",
			Value: bson.M{"owner_id": bson.M{"$not": bson.M{"$in": bans.NoOwnership.KeySlice()}}},
		}},
		{{
			Key:   "$match",
			Value: filter,
		}},
		{{
			Key: "$group",
			Value: bson.M{
				"_id": nil,
				"emotes": bson.M{
					"$push": "$$ROOT",
				},
			},
		}},
		{{
			Key: "$lookup",
			Value: mongo.Lookup{
				From:         mongo.CollectionNameUsers,
				LocalField:   "emotes.owner_id",
				ForeignField: "_id",
				As:           "emote_owners",
			},
		}},
		{{
			Key: "$lookup",
			Value: mongo.Lookup{
				From:         mongo.CollectionNameEntitlements,
				LocalField:   "emote_owners._id",
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
	})
	if err != nil {
		return qr.setError(err)
	}

	cur.Next(ctx)
	v := &aggregatedEmotesResult{}
	if err = cur.Decode(v); err != nil {
		if err == io.EOF {
			return qr.setError(errors.ErrNoItems())
		}
		return qr.setItems(items).setError(err)
	}

	// Map all objects
	qb := &QueryBinder{ctx, q}
	ownerMap, err := qb.MapUsers(v.EmoteOwners, v.RoleEntitlements...)
	if err != nil {
		return qr.setError(err)
	}

	for _, e := range v.Emotes { // iterate over emotes
		// add owner
		if _, banned := bans.MemoryHole[e.OwnerID]; banned {
			e.OwnerID = primitive.NilObjectID
		} else {
			owner := ownerMap[e.OwnerID]
			e.Owner = &owner
		}
		items = append(items, e)
	}
	if err = multierror.Append(err, cur.Close(ctx)).ErrorOrNil(); err != nil {
		qr.setError(err)
	}

	return qr.setItems(items)
}

type aggregatedEmotesResult struct {
	Emotes           []structures.Emote                 `bson:"emotes"`
	EmoteOwners      []structures.User                  `bson:"emote_owners"`
	RoleEntitlements []structures.Entitlement[bson.Raw] `bson:"role_entitlements"`
}
