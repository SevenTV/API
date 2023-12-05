package mutate

import (
	"context"

	"github.com/seventv/common/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (m *Mutate) DeleteUser(ctx context.Context, userID primitive.ObjectID) (int, error) {
	docsDeletedCount := 0
	for _, query := range userDeleteQueries(userID) {
		res, err := m.mongo.Collection(query.collection).DeleteMany(context.TODO(), query.filter)
		if err != nil {
			return 0, err
		}
		docsDeletedCount += int(res.DeletedCount)
	}
	return docsDeletedCount, nil
}

func userDeleteQueries(user primitive.ObjectID) []userDeleteQuery {
	return []userDeleteQuery{
		{mongo.CollectionNameEmoteSets, bson.M{"owner_id": user}},
		{mongo.CollectionNameMessages, bson.M{"author_id": user}},
		{mongo.CollectionNameMessagesRead, bson.M{"author_id": user}},
		{mongo.CollectionNameUserPresences, bson.M{"user_id": user}},
		{mongo.CollectionNameUsers, bson.M{"_id": user}},
		//{mongo.CollectionNameEntitlements, bson.M{"user_id": user}},
	}
}

type userDeleteQuery struct {
	collection mongo.CollectionName
	filter     bson.M
}
