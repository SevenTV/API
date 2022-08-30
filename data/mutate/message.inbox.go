package mutate

import (
	"context"
	"time"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (m *Mutate) SendInboxMessage(ctx context.Context, mb *structures.MessageBuilder[structures.MessageDataInbox], opt SendInboxMessageOptions) error {
	if mb == nil {
		return errors.ErrInternalIncompleteMutation()
	} else if mb.IsTainted() {
		return errors.ErrMutateTaintedObject()
	}

	// Check actor permissions
	actor := opt.Actor
	if actor == nil || actor.ID.IsZero() || !actor.HasPermission(structures.RolePermissionSendMessages) {
		return errors.ErrInsufficientPrivilege()
	}

	// Find recipients
	recipients := []*structures.User{}
	cur, err := m.mongo.Collection(mongo.CollectionNameUsers).Find(ctx, bson.M{
		"$and": func() bson.A {
			a := bson.A{bson.M{"_id": bson.M{"$in": opt.Recipients}}}
			if opt.ConsiderBlockedUsers { // omit blocked users from recipients?
				a = append(a, bson.M{"blocked_user_ids": bson.M{"$not": bson.M{"$eq": actor.ID}}})
			}

			return a
		}(),
	})

	if err != nil {
		return err
	}

	if err = cur.All(ctx, &recipients); err != nil {
		return err
	}

	// Write message to DB
	result, err := m.mongo.Collection(mongo.CollectionNameMessages).InsertOne(ctx, mb.Message)

	if err != nil {
		return err
	}

	var msgID primitive.ObjectID
	switch t := result.InsertedID.(type) {
	case primitive.ObjectID:
		msgID = t
	}

	// Create read states for the recipients
	w := make([]mongo.WriteModel, len(recipients))
	for i, u := range recipients {
		w[i] = &mongo.InsertOneModel{
			Document: &structures.MessageRead{
				MessageID:   msgID,
				Kind:        structures.MessageKindInbox,
				Timestamp:   time.Now(),
				RecipientID: u.ID,
				Read:        false,
			},
		}
	}

	if _, err = m.mongo.Collection(mongo.CollectionNameMessagesRead).BulkWrite(ctx, w); err != nil {
		return err
	}

	mb.Message.ID = msgID
	mb.MarkAsTainted()

	return nil
}

type SendInboxMessageOptions struct {
	Actor                *structures.User
	Recipients           []primitive.ObjectID
	ConsiderBlockedUsers bool
}
