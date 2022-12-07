package presences

import (
	"context"

	"github.com/seventv/api/internal/loaders"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type Instance interface {
	// ChannelPresence returns a PresenceManager for the given actorID, with only Channel presence data.
	ChannelPresence(ctx context.Context, actorID primitive.ObjectID) PresenceManager[structures.UserPresenceDataChannel]
}

type inst struct {
	mongo   mongo.Instance
	loaders loaders.Instance
}

func New(opt Options) Instance {
	return &inst{
		mongo:   opt.Mongo,
		loaders: opt.Loaders,
	}
}

type Options struct {
	Mongo   mongo.Instance
	Loaders loaders.Instance
}

func (p *inst) ChannelPresence(ctx context.Context, actorID primitive.ObjectID) PresenceManager[structures.UserPresenceDataChannel] {
	presences, _ := p.loaders.PresenceByActorID().Load(actorID)

	items := filterPresenceList[structures.UserPresenceDataChannel](presences, structures.UserPresenceKindChannel)

	return &presenceManager[structures.UserPresenceDataChannel]{
		inst:   p,
		kind:   structures.UserPresenceKindChannel,
		userID: actorID,
		items:  items,
	}
}

// filterPresenceList filters the given presence list by the given kind.
func filterPresenceList[T structures.UserPresenceData](items []structures.UserPresence[bson.Raw], kind structures.UserPresenceKind) []structures.UserPresence[T] {
	var (
		pos int
		err error
	)

	result := make([]structures.UserPresence[T], len(items))

	for _, item := range items {
		if item.Kind != kind {
			continue
		}

		result[pos], err = structures.ConvertPresence[T](item)
		if err != nil {
			zap.S().Errorw("failed to convert presence", "error", err)

			continue
		}

		pos++
	}

	if pos < len(result) {
		result = result[:pos]
	}

	return result
}
