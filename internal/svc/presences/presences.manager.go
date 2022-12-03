package presences

import (
	"context"
	"time"

	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type PresenceManager[T structures.UserPresenceData] interface {
	Items() []structures.UserPresence[T]
	Write(ctx context.Context, ttl time.Duration, data T, opt WritePresenceOptions) error
}

type presenceManager[T structures.UserPresenceData] struct {
	inst   *inst
	userID primitive.ObjectID
	kind   structures.UserPresenceKind
	items  []structures.UserPresence[T]
}

// Items implements PresenceManager
func (pm *presenceManager[T]) Items() []structures.UserPresence[T] {
	return pm.items
}

// Write implements PresenceManager
func (pm *presenceManager[T]) Write(ctx context.Context, ttl time.Duration, data T, opt WritePresenceOptions) error {
	result, err := pm.inst.mongo.Collection(mongo.CollectionNameUserPresences).UpdateOne(ctx, bson.M{
		"actor_id": pm.userID,
		"data":     data,
	}, bson.M{"$set": structures.UserPresence[T]{
		UserID:    pm.userID,
		Authentic: opt.Authentic,
		Timestamp: time.Now(),
		TTL:       time.Now().Add(ttl),
		Kind:      pm.kind,
		Data:      data,
	}}, options.Update().SetUpsert(true))
	if err != nil {
		zap.S().Errorw("failed to write presence", "error", err)

		return err
	}

	if result.UpsertedCount > 0 {
		zap.S().Debugw("write presence", "actor_id", pm.userID, "kind", pm.kind)
	}

	return nil
}

type WritePresenceOptions struct {
	Authentic bool
}
