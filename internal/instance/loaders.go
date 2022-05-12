package instance

import (
	"github.com/SevenTV/Common/dataloader"
	"github.com/SevenTV/Common/structures/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Loaders interface {
	UserByID() UserLoaderByID
	UserByUsername() UserLoaderByUsername
	EmoteByID() EmoteLoaderByID
	EmoteSetByID() EmoteSetLoaderByID
	EmoteSetByUserID() BatchEmoteSetLoaderByID
}

type (
	UserLoaderByID          = *dataloader.DataLoader[primitive.ObjectID, structures.User]
	UserLoaderByUsername    = *dataloader.DataLoader[string, structures.User]
	EmoteLoaderByID         = *dataloader.DataLoader[primitive.ObjectID, structures.Emote]
	EmoteSetLoaderByID      = *dataloader.DataLoader[primitive.ObjectID, structures.EmoteSet]
	BatchEmoteSetLoaderByID = *dataloader.DataLoader[primitive.ObjectID, []structures.EmoteSet]
)
