package loaders

import (
	"context"

	"github.com/SevenTV/Common/dataloader"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/seventv/api/global"
	"github.com/seventv/api/rest/rest"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Loaders struct {
	// Emote Loaders
	EmoteByID          *EmoteLoader
	EmotesByEmoteSetID *BatchEmoteLoader

	// User Loaders
	UserByID         *UserLoader
	UserByIdentifier *WildcardUserLoader
}

func New(gCtx global.Context) *Loaders {
	return &Loaders{
		EmoteByID:          emoteByID(gCtx),
		EmotesByEmoteSetID: emotesByEmoteSetID(gCtx),

		UserByID:         userByID(gCtx),
		UserByIdentifier: userByIdentifier(gCtx),
	}
}

func For(ctx context.Context) *Loaders {
	return ctx.Value(string(rest.LoadersKey)).(*Loaders)
}

type (
	EmoteLoader        = dataloader.DataLoader[primitive.ObjectID, *structures.Emote]
	BatchEmoteLoader   = dataloader.DataLoader[primitive.ObjectID, []*structures.Emote]
	UserLoader         = dataloader.DataLoader[primitive.ObjectID, *structures.User]
	WildcardUserLoader = dataloader.DataLoader[string, *structures.User]
)
