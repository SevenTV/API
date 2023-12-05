package mutate

import (
	"context"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (m *Mutate) MergeUserConnections(ctx context.Context, donorUserID, recipientUserID primitive.ObjectID, connectionID string) error {
	// find donor
	donor := structures.User{}
	if err := m.mongo.Collection("users").FindOne(ctx, primitive.M{"_id": donorUserID}).Decode(&donor); err != nil {
		return err
	}

	// get connection from donor
	donorConnection, i := donor.Connections.Get(connectionID)
	if i == -1 {
		return errors.ErrNoItems()
	}

	// push connection to recipient
	if _, err := m.mongo.Collection("users").UpdateOne(ctx, primitive.M{"_id": recipientUserID}, primitive.M{"$push": primitive.M{"connections": donorConnection}}); err != nil {
		return err
	}

	// delete connection from donor
	if _, err := m.mongo.Collection("users").UpdateOne(ctx, primitive.M{"_id": donorUserID}, primitive.M{"$pull": primitive.M{"connections": donorConnection}}); err != nil {
		return err
	}

	return nil
}
