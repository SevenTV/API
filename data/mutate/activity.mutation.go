package mutate

import (
	"context"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.uber.org/zap"
)

func (m *Mutate) EmitActivity(ctx context.Context, ab *structures.ActivityBuilder) error {
	if ab == nil {
		return errors.ErrInternalIncompleteMutation()
	}

	if ab.Activity.Metadata.UserID.IsZero() {
		return errors.ErrInternalIncompleteMutation().SetDetail("Missing User")
	}

	w := []mongo.WriteModel{}

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
