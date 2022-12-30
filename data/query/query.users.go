package query

import (
	"context"
	"time"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
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
	}, options.Aggregate().SetBatchSize(5).SetMaxAwaitTime(time.Second*30))
	if err != nil {
		zap.S().Errorw("failed to create query to aggregate users", "error", err)

		return r.setError(errors.ErrInternalServerError().SetDetail(err.Error()))
	}

	users := []structures.User{}

	if err := cur.All(ctx, &users); err != nil {
		zap.S().Errorw("failed to decode users", "error", err)

		return r.setError(errors.ErrInternalServerError().SetDetail(err.Error()))
	}

	// Map all users
	userMap := map[primitive.ObjectID]structures.User{}
	for _, u := range users {
		userMap[u.ID] = u
	}

	entitlements, err := q.Entitlements(ctx, bson.M{"user_id": bson.M{
		"$in": utils.Map(users, func(x structures.User) primitive.ObjectID {
			return x.ID
		}),
	}}).Items()
	if err != nil {
		return r.setError(err)
	}

	roles, err := q.Roles(ctx, bson.M{})
	if err != nil {
		zap.S().Errorw("failed to fetch roles", "error", err)
		r.setError(err)

		return r
	}

	var defaultRoleID primitive.ObjectID

	roleMap := make(map[primitive.ObjectID]structures.Role)
	for _, r := range roles {
		roleMap[r.ID] = r

		if r.Default {
			defaultRoleID = r.ID
		}
	}

	entMap := make(map[primitive.ObjectID]EntitlementQueryResult)
	for _, e := range entitlements {
		entMap[e.UserID] = e
	}

	for _, u := range userMap {
		ents := entMap[u.ID]

		roleIDs := make(utils.Set[primitive.ObjectID])
		roleIDs.Add(defaultRoleID)
		roleIDs.Fill(append(
			utils.Map(ents.Roles, func(x structures.Entitlement[structures.EntitlementDataRole]) primitive.ObjectID {
				return x.Data.RefID
			}),
			u.RoleIDs...,
		)...)

		for _, roleID := range roleIDs.Values() {
			if role, ok := roleMap[roleID]; ok {
				u.Roles = append(u.Roles, role)
			}
		}

		items = append(items, u)
	}

	return r.setItems(items)
}
