package loaders

import (
	"context"

	"github.com/SevenTV/Common/dataloader"
	"github.com/SevenTV/Common/utils"
	"github.com/seventv/api/global"
	"github.com/seventv/api/gql/v2/gen/model"
)

const LoadersKey = utils.Key("dataloadersv2")

type Loaders struct {
	UserByID       *UserLoader
	UserByUsername *UserLoader
	UserEmotes     *UserEmotesLoader

	EmoteByID *EmoteLoader
}

func New(gCtx global.Context) *Loaders {
	return &Loaders{
		UserByID:       userLoader(gCtx, "_id"),
		UserByUsername: userLoader(gCtx, "username"),
		UserEmotes:     userEmotesLoader(gCtx),

		EmoteByID: emoteByID(gCtx),
	}
}

func For(ctx context.Context) *Loaders {
	return ctx.Value(LoadersKey).(*Loaders)
}

type (
	EmoteLoader      = dataloader.DataLoader[string, *model.Emote]
	UserLoader       = dataloader.DataLoader[string, *model.User]
	UserEmotesLoader = dataloader.DataLoader[string, []*model.Emote]
)
