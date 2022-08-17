package loaders

import (
	"context"

	"github.com/seventv/common/dataloader"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/query"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const LoadersKey = utils.Key("dataloaders")

type Instance interface {
	UserByID() UserLoaderByID
	UserByUsername() UserLoaderByUsername
	UserByConnectionID() UserByConnectionID
	EmoteByID() EmoteLoaderByID
	EmoteByOwnerID() BatchEmoteLoaderByID
	EmoteSetByID() EmoteSetLoaderByID
	EmoteSetByUserID() BatchEmoteSetLoaderByID
}

type inst struct {
	// User Loaders
	userByID           UserLoaderByID
	userByUsername     UserLoaderByUsername
	userByConnectionID UserByConnectionID

	// Emote Loaders
	emoteByID      EmoteLoaderByID
	emoteByOwnerID BatchEmoteLoaderByID

	// Emote Set Loaders
	emoteSetByID EmoteSetLoaderByID

	emoteSetByUserID BatchEmoteSetLoaderByID

	// inst
	mongo mongo.Instance
	redis redis.Instance
	query *query.Query
}

func New(ctx context.Context, mngo mongo.Instance, rdis redis.Instance, quer *query.Query) Instance {
	l := inst{
		query: quer,
		mongo: mngo,
		redis: rdis,
	}

	l.userByID = userLoader[primitive.ObjectID](ctx, l, "_id")
	l.userByUsername = userLoader[string](ctx, l, "username")
	l.userByConnectionID = userLoader[string](ctx, l, "connection.id")
	l.emoteByID = emoteLoader(ctx, l, "versions.id")
	l.emoteByOwnerID = batchEmoteLoader(ctx, l, "owner_id")
	l.emoteSetByID = emoteSetByID(ctx, l)
	l.emoteSetByUserID = emoteSetByUserID(ctx, l)

	return &l
}

func (l inst) UserByID() UserLoaderByID {
	return l.userByID
}

func (l inst) UserByUsername() UserLoaderByUsername {
	return l.userByUsername
}

func (l inst) UserByConnectionID() UserByConnectionID {
	return l.userByConnectionID
}

func (l inst) EmoteByID() EmoteLoaderByID {
	return l.emoteByID
}

func (l inst) EmoteSetByID() EmoteSetLoaderByID {
	return l.emoteSetByID
}

func (l inst) EmoteSetByUserID() BatchEmoteSetLoaderByID {
	return l.emoteSetByUserID
}

// EmoteByOwnerID implements Instance
func (l *inst) EmoteByOwnerID() BatchEmoteLoaderByID {
	return l.emoteByOwnerID
}

type (
	UserLoaderByID          = *dataloader.DataLoader[primitive.ObjectID, structures.User]
	UserLoaderByUsername    = *dataloader.DataLoader[string, structures.User]
	UserByConnectionID      = *dataloader.DataLoader[string, structures.User]
	EmoteLoaderByID         = *dataloader.DataLoader[primitive.ObjectID, structures.Emote]
	BatchEmoteLoaderByID    = *dataloader.DataLoader[primitive.ObjectID, []structures.Emote]
	EmoteSetLoaderByID      = *dataloader.DataLoader[primitive.ObjectID, structures.EmoteSet]
	BatchEmoteSetLoaderByID = *dataloader.DataLoader[primitive.ObjectID, []structures.EmoteSet]
)
