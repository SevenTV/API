package loaders

import (
	"github.com/SevenTV/Common/utils"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/instance"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const LoadersKey = utils.Key("dataloaders")

type Loaders struct {
	// User Loaders
	userByID       instance.UserLoaderByID
	userByUsername instance.UserLoaderByUsername

	// Emote Loaders
	emoteByID instance.EmoteLoaderByID

	// Emote Set Loaders
	emoteSetByID instance.EmoteSetLoaderByID

	emoteSetByUserID instance.BatchEmoteSetLoaderByID
}

func New(gCtx global.Context) instance.Loaders {
	return &Loaders{
		userByID:         userLoader[primitive.ObjectID](gCtx, "_id"),
		userByUsername:   userLoader[string](gCtx, "username"),
		emoteByID:        emoteByID(gCtx),
		emoteSetByID:     emoteSetByID(gCtx),
		emoteSetByUserID: emoteSetByUserID(gCtx),
	}
}

func (l Loaders) UserByID() instance.UserLoaderByID {
	return l.userByID
}

func (l Loaders) UserByUsername() instance.UserLoaderByUsername {
	return l.userByUsername
}

func (l Loaders) EmoteByID() instance.EmoteLoaderByID {
	return l.emoteByID
}

func (l Loaders) EmoteSetByID() instance.EmoteSetLoaderByID {
	return l.emoteSetByID
}

func (l Loaders) EmoteSetByUserID() instance.BatchEmoteSetLoaderByID {
	return l.emoteSetByUserID
}
