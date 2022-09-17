package query

import (
	"context"
	"time"

	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const ONLINE_ACTIVITY_TIMEOUT = 5 * time.Minute

func (r *Resolver) OnlineUsers(ctx context.Context) ([]*model.UserPartial, error) {
	roles, err := r.Ctx.Inst().Query.Roles(ctx, bson.M{
		"staff": true,
	})
	if err != nil {
		r.Z().Errorw("failed to query roles", "error", err)

		return nil, errors.ErrInternalServerError()
	}

	roleIDs := make([]primitive.ObjectID, len(roles))
	for i, role := range roles {
		roleIDs[i] = role.ID
	}

	// Query staff users
	users, err := r.Ctx.Inst().Query.Users(ctx, bson.M{
		"role_ids": bson.M{"$in": roleIDs},
	}).Items()
	if err != nil {
		r.Z().Errorw("failed to query users", "error", err)

		return nil, errors.ErrInternalServerError()
	}

	userMap := make(map[primitive.ObjectID]structures.User)
	userIDs := make([]primitive.ObjectID, len(users))

	for i, user := range users {
		userIDs[i] = user.ID
		userMap[user.ID] = user
	}

	cur, _ := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameActivities).Aggregate(ctx, mongo.Pipeline{
		{{
			Key: "$match",
			Value: bson.M{
				"values.status":    bson.M{"$in": []structures.ActivityStatus{structures.ActivityStatusOnline, structures.ActivityStatusIdle, structures.ActivityStatusDnd}},
				"metadata.user_id": bson.M{"$in": userIDs},
				"timestamp": bson.M{
					"$gte": time.Now().Add(-ONLINE_ACTIVITY_TIMEOUT),
				},
			},
		}},
		{{
			Key: "$group",
			Value: bson.M{
				"_id": "$metadata.user_id",
				"activity": bson.M{
					"$last": "$$ROOT",
				},
			},
		}},
	})

	result := []*model.UserPartial{}

	for cur.Next(ctx) {
		a := aggregatedLatestActivity{}

		if err := cur.Decode(&a); err != nil {
			r.Z().Errorw("failed to decode activity", "error", err)

			continue
		}

		user, ok := userMap[a.Activity.Metadata.UserID]
		if !ok {
			continue
		}

		result = append(result, helpers.UserStructureToPartialModel(helpers.UserStructureToModel(user, r.Ctx.Config().CdnURL)))
	}

	return result, nil
}

type aggregatedLatestActivity struct {
	Activity structures.Activity `bson:"activity"`
}
