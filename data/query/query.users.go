package query

import (
	"context"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func (q *Query) Users(ctx context.Context, filter bson.M) *QueryResult[structures.User] {
	items := []structures.User{}
	r := &QueryResult[structures.User]{}

	cur, err := q.mongo.Collection(mongo.CollectionNameUsers).Find(ctx, filter)
	if err != nil {
		return r.setError(errors.ErrInternalServerError().SetDetail(err.Error()))
	}

	// Get roles
	roleMap := make(map[primitive.ObjectID]structures.Role)
	roles, _ := q.Roles(ctx, bson.M{})

	for _, role := range roles {
		roleMap[role.ID] = role
	}

	for i := 0; cur.Next(ctx); i++ {
		item := structures.User{}
		if err = cur.Decode(&item); err != nil {
			zap.S().Errorw("failed to decode user", "error", err)
		}

		items = append(items, item)
	}

	return r.setItems(items)
}
