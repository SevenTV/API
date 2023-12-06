package mutate

import (
	"context"

	"github.com/seventv/common/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (m *Mutate) DeleteUser(ctx context.Context, userID primitive.ObjectID) (int, error) {
	docsDeletedCount := 0

	// Delete all EUD
	for _, query := range userDeleteQueries(userID) {
		res, err := m.mongo.Collection(query.collection).DeleteMany(ctx, query.filter)
		if err != nil {
			return 0, err
		}
		docsDeletedCount += int(res.DeletedCount)
	}

	return docsDeletedCount, nil
}

func userDeleteQueries(userID primitive.ObjectID) []userDeleteQuery {
	return []userDeleteQuery{
		{mongo.CollectionNameEmoteSets, bson.M{"owner_id": userID}},
		{mongo.CollectionNameMessages, bson.M{"author_id": userID}},
		{mongo.CollectionNameMessagesRead, bson.M{"author_id": userID}},
		{mongo.CollectionNameUserPresences, bson.M{"user_id": userID}},
		{mongo.CollectionNameUsers, bson.M{"_id": userID}},
		// {mongo.CollectionNameEntitlements, bson.M{"user_id": user}},
	}
}

type userDeleteQuery struct {
	collection mongo.CollectionName
	filter     bson.M
}
