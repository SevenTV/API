package presences

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/data/events"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (p *inst) ChannelPresenceFanout(ctx context.Context, presence structures.UserPresence[structures.UserPresenceDataChannel]) error {
	// Fetch user of the presence
	_, err := p.loaders.UserByID().Load(presence.UserID)
	if err != nil {
		return err
	}

	eventCond := events.EventCondition{
		"ctx":      "channel",
		"platform": string(presence.Data.Platform),
		"id":       presence.Data.ID,
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

	// Dispatch personal emote sets
	{
		entMap := make(map[primitive.ObjectID]structures.Entitlement[structures.EntitlementDataEmoteSet])
		setIDs := make([]primitive.ObjectID, len(cosmetics.EmoteSets))

		for i, ent := range cosmetics.EmoteSets {
			setIDs[i] = ent.Data.RefID
			entMap[ent.Data.RefID] = ent
		}

		// Fetch Emote Sets
		sets, errs := p.loaders.EmoteSetByID().LoadAll(setIDs)
		if multierror.Append(nil, errs...).ErrorOrNil() != nil {
			return err
		}

		for _, es := range sets {
			es.Owner = nil

			ent, ok := entMap[es.ID]
			if !ok {
				continue // can't find linked entitlement
			}

			// Dispatch the Emote Set data
			_ = p.events.Dispatch(ctx, events.EventTypeCreateEmoteSet, events.ChangeMap{
				ID:         es.ID,
				Kind:       structures.ObjectKindEmoteSet,
				Contextual: true,
				Object:     utils.ToJSON(p.modelizer.EmoteSet(es)),
			}, eventCond)

			// Dispatch the Emote Set entitlement
			_ = p.events.Dispatch(ctx, events.EventTypeCreateEntitlement, events.ChangeMap{
				ID:         ent.ID,
				Kind:       structures.ObjectKindEntitlement,
				Contextual: true,
				Object:     utils.ToJSON(p.modelizer.Entitlement(ent.ToRaw(), user)),
			}, eventCond)
		}
	}

	return nil
}
