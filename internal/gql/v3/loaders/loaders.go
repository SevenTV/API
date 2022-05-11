package loaders

import (
	"context"

	"github.com/SevenTV/Common/dataloader"
	"github.com/SevenTV/Common/utils"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const LoadersKey = utils.Key("dataloaders")

type Loaders struct {
	// User Loaders
	UserByID *UserLoader

	// Emote Loaders
	EmoteByID *EmoteLoader

	// Emote Set Loaders
	EmoteSetByID     *EmoteSetLoader
	EmoteSetByUserID *BatchEmoteSetLoader
}

func New(gCtx global.Context) *Loaders {
	return &Loaders{
		UserByID:         userByID(gCtx),
		EmoteByID:        emoteByID(gCtx),
		EmoteSetByID:     emoteSetByID(gCtx),
		EmoteSetByUserID: emoteSetByUserID(gCtx),
	}
}

func For(ctx context.Context) *Loaders {
	return ctx.Value(LoadersKey).(*Loaders)
}

type (
	EmoteLoader         = dataloader.DataLoader[primitive.ObjectID, *model.Emote]
	UserLoader          = dataloader.DataLoader[primitive.ObjectID, *model.User]
	BatchUserLoader     = dataloader.DataLoader[primitive.ObjectID, []*model.User]
	EmoteSetLoader      = dataloader.DataLoader[primitive.ObjectID, *model.EmoteSet]
	BatchEmoteSetLoader = dataloader.DataLoader[primitive.ObjectID, []*model.EmoteSet]
)
