package presences

import (
	"context"

	"github.com/seventv/api/data/events"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
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

	// Fetch user's active cosmetics
	cosmetics, err := p.loaders.EntitlementsLoader().Load(presence.UserID)
	if err != nil {
		return err
	}

	// Dispatch badge
	badge, badgeEnt, hasBadge := cosmetics.ActiveBadge()
	if hasBadge {
		// Badge Cosmetic
		_ = p.events.Dispatch(ctx, events.EventTypeCreateCosmetic, events.ChangeMap{
			ID:         badge.ID,
			Kind:       structures.ObjectKindCosmetic,
			Contextual: true,
			Object:     utils.ToJSON(p.modelizer.Cosmetic(badge.ToRaw())),
		}, eventCond)

		// Badge Entitlement
		_ = p.events.Dispatch(ctx, events.EventTypeCreateEntitlement, events.ChangeMap{
			ID:         badgeEnt.ID,
			Kind:       structures.ObjectKindEntitlement,
			Contextual: true,
			Object:     utils.ToJSON(p.modelizer.Entitlement(badgeEnt.ToRaw())),
		}, eventCond)
	}

	// Dispatch paint
	paint, paintEnt, hasPaint := cosmetics.ActivePaint()
	if hasPaint {
		// Paint Cosmetic
		_ = p.events.Dispatch(ctx, events.EventTypeCreateCosmetic, events.ChangeMap{
			ID:         paint.ID,
			Kind:       structures.ObjectKindCosmetic,
			Contextual: true,
			Object:     utils.ToJSON(p.modelizer.Cosmetic(paint.ToRaw())),
		}, eventCond)

		// Paint Entitlement
		_ = p.events.Dispatch(ctx, events.EventTypeCreateEntitlement, events.ChangeMap{
			ID:         paintEnt.ID,
			Kind:       structures.ObjectKindEntitlement,
			Contextual: true,
			Object:     utils.ToJSON(p.modelizer.Entitlement(paintEnt.ToRaw())),
		}, eventCond)
	}

	// Dispatch entitlements

	return nil
}
