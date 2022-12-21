package presences

import (
	"context"

	"github.com/seventv/api/data/events"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
)

func (p *inst) ChannelPresenceFanout(ctx context.Context, presence structures.UserPresence[structures.UserPresenceDataChannel]) error {
	// Fetch user of the presence
	_, err := p.loaders.UserByID().Load(presence.UserID)
	if err != nil {
		return err
	}

	eventCond := events.EventCondition{
		"host_id":       presence.Data.HostID.Hex(),
		"connection_id": presence.Data.ConnectionID,
	}

	// Fetch user in presence
	user, err := p.loaders.UserByID().Load(presence.UserID)
	if err != nil {
		return err
	}

	// Fetch user's active cosmetics
	cosmetics, err := p.loaders.EntitlementsLoader().Load(presence.UserID)
	if err != nil {
		return err
	}

	dispatchCosmetic := func(cos structures.Cosmetic[bson.Raw], ent structures.Entitlement[bson.Raw]) {
		// Cosmetic
		_ = p.events.Dispatch(ctx, events.EventTypeCreateCosmetic, events.ChangeMap{
			ID:         cos.ID,
			Kind:       structures.ObjectKindCosmetic,
			Contextual: true,
			Object:     utils.ToJSON(p.modelizer.Cosmetic(cos.ToRaw())),
		}, eventCond)

		// Entitlement
		_ = p.events.Dispatch(ctx, events.EventTypeCreateEntitlement, events.ChangeMap{
			ID:         ent.ID,
			Kind:       structures.ObjectKindEntitlement,
			Contextual: true,
			Object:     utils.ToJSON(p.modelizer.Entitlement(ent.ToRaw(), user)),
		}, eventCond)
	}

	// Dispatch badge
	if badge, badgeEnt, hasBadge := cosmetics.ActiveBadge(); hasBadge {
		dispatchCosmetic(badge.ToRaw(), badgeEnt.ToRaw())
	}

	// Dispatch paint
	if paint, paintEnt, hasPaint := cosmetics.ActivePaint(); hasPaint {
		dispatchCosmetic(paint.ToRaw(), paintEnt.ToRaw())
	}

	// TODO: dispatch personal emote sets

	return nil
}
