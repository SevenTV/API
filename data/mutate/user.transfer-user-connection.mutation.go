package mutate

import (
	"context"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func (m *Mutate) TransferUserConnection(ctx context.Context, actor structures.User, transferer, transferee structures.User, connectionID string) error {
	// Check permissions
	if (!actor.ID.IsZero() && actor.ID != transferer.ID) && actor.GetHighestRole().Position <= transferer.GetHighestRole().Position {
		return errors.ErrInsufficientPrivilege().SetDetail("Lower than victim")
	}

	// Get connection from the outgoing user
	connection, i := transferer.Connections.Get(connectionID)
	if i == -1 {
		return errors.ErrUnknownUserConnection()
	}

	// push connection to recipient
	if _, err := m.mongo.Collection("users").UpdateOne(ctx, primitive.M{"_id": transferee.ID}, primitive.M{"$push": primitive.M{"connections": connection}}); err != nil {
		zap.S().Errorw("mutate, TransferUserConnection(), couldn't push connection to transferee")

		return errors.ErrInternalServerError()
	}

	// delete connection from donor
	if _, err := m.mongo.Collection("users").UpdateOne(ctx, primitive.M{"_id": transferer.ID}, primitive.M{"$pull": primitive.M{"connections": connection}}); err != nil {
		zap.S().Errorw("mutate, TransferUserConnection(), couldn't pull connection from transferer")

		return errors.ErrInternalServerError()
	}

	return nil
}
