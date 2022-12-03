package presences

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

const (
	MOST_UNIQUE_PRESENCES_PER_IP = 3
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
	// Perform protective measures in the case of an unauthentic presence
	if !opt.Authentic {
		cur, err := pm.inst.mongo.Collection(mongo.CollectionNameUserPresences).Find(ctx, bson.M{
			"kind": pm.kind,
			"ip":   opt.IP,
		}, options.Find().SetProjection(bson.M{"user_id": 1}))

		if err == nil {
			// Check the user IDs this IP is occupying with unauthentic presences
			userIDs := utils.Set[primitive.ObjectID]{}

			for cur.Next(ctx) {
				item := structures.UserPresence[T]{}

				if err := cur.Decode(&item); err != nil {
					continue
				}

				userIDs.Add(item.UserID)
			}

			// Decline if this IP has issued too many unauthentic presences over different users
			//
			// This measure prevents a malicious actor from "spoofing" many users
			// and causing unnecessary extra data to be delivered to listeners.
			if len(userIDs) >= MOST_UNIQUE_PRESENCES_PER_IP {
				return errors.ErrRateLimited().SetDetail("Too Many Unauthentic Presences")
			}
		}
	}

	// Write the presence
	result, err := pm.inst.mongo.Collection(mongo.CollectionNameUserPresences).UpdateOne(ctx, bson.M{
		"actor_id": pm.userID,
		"data":     data,
	}, bson.M{"$set": structures.UserPresence[T]{
		UserID:    pm.userID,
		IP:        opt.IP,
		Authentic: opt.Authentic,
		Timestamp: time.Now(),
		TTL:       time.Now().Add(ttl),
		Kind:      pm.kind,
		Data:      data,
	}}, options.Update().SetUpsert(true))
	if err != nil {
		zap.S().Errorw("failed to write presence", "error", err)

		return errors.ErrInternalServerError()
	}

	if result.UpsertedCount > 0 {
		zap.S().Debugw("write presence", "actor_id", pm.userID, "kind", pm.kind)
	}

	return nil
}

type WritePresenceOptions struct {
	Authentic bool
	IP        string
}
