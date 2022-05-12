package emote

import (
	"context"

	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/api/internal/gql/v3/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.EmoteResolver {
	return &Resolver{r}
}

func (r *Resolver) Owner(ctx context.Context, obj *model.Emote) (*model.User, error) {
	if obj.Owner != nil && obj.Owner.ID != structures.DeletedUser.ID {
		return obj.Owner, nil
	}

	user, err := r.Ctx.Inst().Loaders.UserByID().Load(obj.OwnerID)
	if err != nil && !errors.Compare(err, errors.ErrUnknownUser()) {
		return nil, err
	}

	return helpers.UserStructureToModel(user, r.Ctx.Config().CdnURL), nil
}

func (r *Resolver) ChannelCount(ctx context.Context, obj *model.Emote) (int, error) {
	count, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).CountDocuments(ctx, bson.M{
		"channel_emotes.id": obj.ID,
	})
	if err != nil {
		zap.S().Errorw("failed to count documents for emotes",
			"err", err,
		)
		return 0, err
	}

	return int(count), nil
}

func (r *Resolver) Reports(ctx context.Context, obj *model.Emote) ([]*model.Report, error) {
	// return loaders.For(ctx).ReportsByEmoteID.Load(obj.ID)
	return nil, nil
}
