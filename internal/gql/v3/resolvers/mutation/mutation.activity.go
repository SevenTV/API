package mutation

import (
	"context"
	"time"

	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) SendActivity(ctx context.Context, status model.ActivityStatus, typ *model.ActivityTypeInput, obj *model.ActivityObjectInput) (primitive.ObjectID, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return primitive.NilObjectID, errors.ErrUnauthorized()
	}

	typeMap := map[model.ActivityType]structures.ActivityType{
		model.ActivityTypeViewing:   structures.ActivityTypeViewing,
		model.ActivityTypeEditing:   structures.ActivityTypeEditing,
		model.ActivityTypeWatching:  structures.ActivityTypeWatching,
		model.ActivityTypeListening: structures.ActivityTypeListening,
		model.ActivityTypeChatting:  structures.ActivityTypeChatting,
		model.ActivityTypeCreating:  structures.ActivityTypeCreating,
		model.ActivityTypeUpdating:  structures.ActivityTypeUpdating,
	}

	var (
		aname string
		atype structures.ActivityType
	)

	if typ != nil {
		aname = typ.Name
		atype = typeMap[typ.Type]
	}

	astatus := map[model.ActivityStatus]structures.ActivityStatus{
		model.ActivityStatusOffline: structures.ActivityStatusOffline,
		model.ActivityStatusIDLe:    structures.ActivityStatusIdle,
		model.ActivityStatusDnd:     structures.ActivityStatusDnd,
		model.ActivityStatusOnline:  structures.ActivityStatusOnline,
	}[status]

	id := primitive.NewObjectIDFromTimestamp(time.Now())

	ab := structures.NewActivityBuilder(structures.Activity{
		ID:        id,
		Timestamp: time.Now(),
	})

	ab.SetUserID(actor.ID)
	ab.SetType(atype)
	ab.SetName(structures.ActivityName(aname))
	ab.SetStatus(astatus)
	ab.SetTimespan(time.Now(), time.Time{})

	if obj != nil {
		ab.SetObject(structures.ObjectKind(obj.TargetKind), obj.TargetID)
	}

	if err := r.Ctx.Inst().Mutate.EmitActivity(ctx, ab); err != nil {
		return primitive.NilObjectID, err
	}

	return id, nil
}
