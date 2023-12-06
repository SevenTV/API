package mutate

import (
	"context"
	"fmt"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func (m *Mutate) DeleteUser(ctx context.Context, opt DeleteUserOptions) (int, error) {
	docsDeletedCount := 0

	if opt.Victim.ID.IsZero() || opt.Actor.ID.IsZero() {
		return 0, errors.ErrInternalIncompleteMutation()
	}

	if opt.Actor.GetHighestRole().Position <= opt.Victim.GetHighestRole().Position {
		return 0, errors.ErrInsufficientPrivilege()
	}
	fmt.Println(opt.Victim, opt.Actor)

	// Delete all EUD
	for _, query := range userDeleteQueries(opt.Victim.ID) {
		res, err := m.mongo.Collection(query.collection).DeleteMany(ctx, query.filter)
		if err != nil {
			zap.S().Errorw("mutate, DeleteUser()", "error", err)

			return 0, err
		}

		docsDeletedCount += int(res.DeletedCount)
	}

	// Delete editor references
	if _, err := m.mongo.Collection(mongo.CollectionNameUsers).UpdateMany(ctx, bson.M{
		"editors.id": opt.Victim.ID,
	}, bson.M{
		"$pull": bson.M{"editors": bson.M{"id": opt.Victim.ID}},
	}); err != nil {
		zap.S().Errorw("mutate, DeleteUser(), failed to remove editor references", "error", err)
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

type DeleteUserOptions struct {
	Actor  structures.User
	Victim structures.User
}
