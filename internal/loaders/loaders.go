package loaders

import (
	"context"

	"github.com/seventv/api/data/query"
	"github.com/seventv/common/dataloader"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const LoadersKey = utils.Key("dataloaders")

type Instance interface {
	UserByID() UserLoaderByID
	UserByUsername() UserLoaderByUsername
	UserByConnectionID(structures.UserConnectionPlatform) UserLoaderByConnectionID
	UserByConnectionUsername(structures.UserConnectionPlatform) UserLoaderByConnectionUsername

	EmoteByID() EmoteLoaderByID
	EmoteByOwnerID() BatchEmoteLoaderByID
	EmoteSetByID() EmoteSetLoaderByID
	EmoteSetByUserID() BatchEmoteSetLoaderByID

	PresenceByActorID() PresenceLoaderByID

	EntitlementsLoader() EntitlementsLoader
}

type inst struct {
	// User Loaders
	userByID                 UserLoaderByID
	userByUsername           UserLoaderByUsername
	userByConnectionID       map[structures.UserConnectionPlatform]UserLoaderByConnectionID
	userByConnectionUsername map[structures.UserConnectionPlatform]UserLoaderByConnectionUsername

	// Emote Loaders
	emoteByID      EmoteLoaderByID
	emoteByOwnerID BatchEmoteLoaderByID

	// Emote Set Loaders
	emoteSetByID     EmoteSetLoaderByID
	emoteSetByUserID BatchEmoteSetLoaderByID

	// Presence Loaders
	presenceByActorID PresenceLoaderByID

	// Entitlements
	entitlements EntitlementsLoader

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
	l.userByConnectionID = map[structures.UserConnectionPlatform]*dataloader.DataLoader[string, structures.User]{
		structures.UserConnectionPlatformTwitch:  userByConnectionLoader(ctx, l, structures.UserConnectionPlatformTwitch, "id"),
		structures.UserConnectionPlatformYouTube: userByConnectionLoader(ctx, l, structures.UserConnectionPlatformYouTube, "id"),
		structures.UserConnectionPlatformDiscord: userByConnectionLoader(ctx, l, structures.UserConnectionPlatformDiscord, "id"),
	}
	l.userByConnectionUsername = map[structures.UserConnectionPlatform]*dataloader.DataLoader[string, structures.User]{
		structures.UserConnectionPlatformTwitch:  userByConnectionLoader(ctx, l, structures.UserConnectionPlatformTwitch, "data.login"),
		structures.UserConnectionPlatformYouTube: userByConnectionLoader(ctx, l, structures.UserConnectionPlatformYouTube, "data.username"),
		structures.UserConnectionPlatformDiscord: userByConnectionLoader(ctx, l, structures.UserConnectionPlatformDiscord, "data.username"),
	}

	l.emoteByID = emoteLoader(ctx, l, "versions.id")
	l.emoteByOwnerID = batchEmoteLoader(ctx, l, "owner_id")
	l.emoteSetByID = emoteSetByID(ctx, l)
	l.emoteSetByUserID = emoteSetByUserID(ctx, l)

	l.presenceByActorID = presenceLoader[bson.Raw](ctx, l, structures.UserPresenceKindUnknown, "actor_id")

	l.entitlements = entitlementsLoader(ctx, l)

	return &l
}

func (l inst) UserByID() UserLoaderByID {
	return l.userByID
}

func (l inst) UserByUsername() UserLoaderByUsername {
	return l.userByUsername
}

func (l inst) UserByConnectionID(platform structures.UserConnectionPlatform) UserLoaderByConnectionID {
	loader, ok := l.userByConnectionID[platform]
	if !ok {
		return l.userByConnectionID[structures.UserConnectionPlatformTwitch]
	}

	return loader
}

func (l inst) UserByConnectionUsername(platform structures.UserConnectionPlatform) UserLoaderByConnectionUsername {
	loader, ok := l.userByConnectionUsername[platform]
	if !ok {
		return l.userByConnectionUsername[structures.UserConnectionPlatformTwitch]
	}

	return loader
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

func (l *inst) PresenceByActorID() PresenceLoaderByID {
	return l.presenceByActorID
}

func (l *inst) EntitlementsLoader() EntitlementsLoader {
	return l.entitlements
}

type (
	UserLoaderByID                 = *dataloader.DataLoader[primitive.ObjectID, structures.User]
	UserLoaderByUsername           = *dataloader.DataLoader[string, structures.User]
	UserLoaderByConnectionID       = *dataloader.DataLoader[string, structures.User]
	UserLoaderByConnectionUsername = *dataloader.DataLoader[string, structures.User]

	EmoteLoaderByID         = *dataloader.DataLoader[primitive.ObjectID, structures.Emote]
	BatchEmoteLoaderByID    = *dataloader.DataLoader[primitive.ObjectID, []structures.Emote]
	EmoteSetLoaderByID      = *dataloader.DataLoader[primitive.ObjectID, structures.EmoteSet]
	BatchEmoteSetLoaderByID = *dataloader.DataLoader[primitive.ObjectID, []structures.EmoteSet]

	PresenceLoaderByID = *dataloader.DataLoader[primitive.ObjectID, []structures.UserPresence[bson.Raw]]

	EntitlementsLoader = *dataloader.DataLoader[primitive.ObjectID, query.EntitlementQueryResult]
)
