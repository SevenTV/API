package loaders

import (
	"context"

	"github.com/SevenTV/Common/dataloader"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/Common/utils"
	"github.com/seventv/api/internal/global"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const LoadersKey = utils.Key("dataloadersv2")

type Loaders struct {
	UserByID       UserLoaderByID
	UserByUsername UserLoaderByUsername
	UserEmotes     UserEmotesLoader
	EmoteByID      EmoteLoader
}

func New(gCtx global.Context) *Loaders {
	return &Loaders{
		UserByID:       userLoader[primitive.ObjectID](gCtx, "_id"),
		UserByUsername: userLoader[string](gCtx, "username"),
		UserEmotes:     userEmotesLoader(gCtx),
		EmoteByID:      emoteByID(gCtx),
	}
}

func For(ctx context.Context) *Loaders {
	return ctx.Value(LoadersKey).(*Loaders)
}

type (
	EmoteLoader          = *dataloader.DataLoader[primitive.ObjectID, structures.Emote]
	UserLoaderByID       = *dataloader.DataLoader[primitive.ObjectID, structures.User]
	UserLoaderByUsername = *dataloader.DataLoader[string, structures.User]
	UserEmotesLoader     = *dataloader.DataLoader[string, []structures.ActiveEmote]
)
