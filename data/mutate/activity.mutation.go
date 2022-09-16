package mutate

import (
	"context"
	"time"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

func (m *Mutate) EmitActivity(ctx context.Context, ab *structures.ActivityBuilder) error {
	if ab == nil {
		return errors.ErrInternalIncompleteMutation()
	}

	if ab.Activity.State.UserID.IsZero() {
		return errors.ErrInternalIncompleteMutation().SetDetail("Missing User")
	}

	w := []mongo.WriteModel{}

	// Update end dates of past activities
	w = append(w, &mongo.UpdateManyModel{
		Filter: bson.M{
			"state.user_id":      ab.Activity.State.UserID,
			"state.timespan.end": nil,
		},
		Update: bson.M{"$set": bson.M{
			"state.timespan.end": time.Now(),
		}},
	})

	// Insert new activity
	w = append(w, &mongo.InsertOneModel{
		Document: ab.Activity,
	})

	if _, err := m.mongo.Collection(mongo.CollectionNameActivities).BulkWrite(ctx, w); err != nil {
		zap.S().Errorw("mongo, error while writing new activity", "error", err)

		return errors.ErrInternalServerError()
	}

	return nil
}
