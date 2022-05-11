package emote

import (
	"context"

	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/seventv/api/gql/v3/gen/generated"
	"github.com/seventv/api/gql/v3/gen/model"
	"github.com/seventv/api/gql/v3/loaders"
	"github.com/seventv/api/gql/v3/types"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
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
	return loaders.For(ctx).UserByID.Load(obj.OwnerID)
}

func (r *Resolver) ChannelCount(ctx context.Context, obj *model.Emote) (int, error) {
	count, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).CountDocuments(ctx, bson.M{
		"channel_emotes.id": obj.ID,
	})
	if err != nil {
		logrus.WithError(err).Error("failed to count documents for emotes")
		return 0, err
	}

	return int(count), nil
}

func (r *Resolver) Reports(ctx context.Context, obj *model.Emote) ([]*model.Report, error) {
	// return loaders.For(ctx).ReportsByEmoteID.Load(obj.ID)
	return nil, nil
}
