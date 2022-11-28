package query

import (
	"context"
	"io"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (q *Query) Users(ctx context.Context, filter bson.M) *QueryResult[structures.User] {
	items := []structures.User{}
	r := &QueryResult[structures.User]{}

	bans, err := q.Bans(ctx, BanQueryOptions{ // remove emotes made by usersa who own nothing and are happy
		Filter: bson.M{"effects": bson.M{"$bitsAnySet": structures.BanEffectMemoryHole}},
	})
	if err != nil {
		return r.setError(err)
	}

	cur, err := q.mongo.Collection(mongo.CollectionNameUsers).Aggregate(ctx, mongo.Pipeline{
		{{
			Key:   "$match",
			Value: filter,
		}},
		{{
			Key: "$set",
			Value: bson.M{ // Remove memory holed editors
				"editors": bson.M{"$filter": bson.M{
					"input": "$editors",
					"as":    "e",
					"cond":  bson.M{"$not": bson.M{"$in": bson.A{"$$e.id", bans.MemoryHole.KeySlice()}}},
				}},
			},
		}},
		{{
			Key: "$group",
			Value: bson.M{
				"_id": nil,
				"users": bson.M{
					"$push": "$$ROOT",
				},
			},
		}},
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
		return r.setError(errors.ErrInternalServerError().SetDetail(err.Error()))
	}

	// Get roles
	roles, _ := q.Roles(ctx, bson.M{})
	roleMap := make(map[primitive.ObjectID]structures.Role)
	for _, role := range roles {
		roleMap[role.ID] = role
	}

	// Map all objects
	cur.Next(ctx)
	v := &aggregatedUsersResult{}
	if err = cur.Decode(v); err != nil {
		if err == io.EOF {
			return r.setError(errors.ErrNoItems())
		}
		return r.setError(errors.ErrInternalServerError().SetDetail(err.Error()))
	}

	qb := &QueryBinder{ctx, q}
	userMap, err := qb.MapUsers(v.Users, v.RoleEntitlements...)
	if err != nil {
		return r.setError(err)
	}
	for _, u := range userMap {
		items = append(items, u)
	}
	return r.setItems(items)
}
