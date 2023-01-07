package presences

import (
	"context"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/data/events"
	"github.com/seventv/api/data/query"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func (p *inst) ChannelPresenceFanout(ctx context.Context, presence structures.UserPresence[structures.UserPresenceDataChannel]) error {
	eventCond := events.EventCondition{
		"ctx":      "channel",
		"platform": string(presence.Data.Platform),
		"id":       presence.Data.ID,
	}

	entEventCond := events.EventCondition{
		"user_id": presence.UserID.Hex(),
	}

	var (
		user         structures.User
		cosmetics    query.EntitlementQueryResult
		entitlements []structures.UserPresenceEntitlement
		msg          events.Message[events.DispatchPayload]
		err          error
	)

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		// Fetch user in presence
		user, err = p.loaders.UserByID().Load(presence.UserID)

		wg.Done()
	}()

	go func() {
		// Fetch user's active cosmetics
		cosmetics, err = p.loaders.EntitlementsLoader().Load(presence.UserID)

		wg.Done()
	}()

	wg.Wait()

	if err != nil {
		return err
	}

	previousHashList := make(utils.Set[uint32])
	previousHashMap := make(map[primitive.ObjectID]uint32)

	for _, ent := range presence.Entitlements {
		previousHashMap[ent.RefID] = ent.DispatchHash

		if ent.DispatchHash > 0 {
			previousHashList.Add(ent.DispatchHash)
		}
	}

	dispatchCosmetic := func(cos structures.Cosmetic[bson.Raw], ent structures.Entitlement[structures.EntitlementDataBase]) {
		// Cosmetic
		_ = p.events.Dispatch(ctx, events.EventTypeCreateCosmetic, events.ChangeMap{
			ID:     cos.ID,
			Kind:   structures.ObjectKindCosmetic,
			Object: utils.ToJSON(p.modelizer.Cosmetic(cos.ToRaw())),
		}, eventCond)

		// Check dispatch hash map of previous entitlements
		//
		// If not present, it will be expired in the event api's dedupe cache
		if ha, ok := previousHashMap[ent.Data.RefID]; ok {
			previousHashList.Delete(ha)
		}

		// Dispatch: Entitlement
		msg, _ = p.events.DispatchWithEffect(ctx, events.EventTypeCreateEntitlement, events.ChangeMap{
			ID:         ent.ID,
			Kind:       structures.ObjectKindEntitlement,
			Contextual: true,
			Object:     utils.ToJSON(p.modelizer.Entitlement(ent.ToRaw(), user)),
		}, &events.SessionEffect{
			AddSubscriptions: []events.SubscribePayload{{
				Type:      events.EventTypeAnyEntitlement,
				Condition: map[string]string{"user_id": presence.UserID.Hex()},
			}},
			RemoveHashes: previousHashList.Values(),
		}, eventCond, entEventCond)

		// Add to presence's entitlement refs
		entRef := structures.UserPresenceEntitlement{
			Kind:  ent.Kind,
			ID:    ent.ID,
			RefID: ent.Data.RefID,
		}
		if msg.Data.Hash != nil {
			entRef.DispatchHash = *msg.Data.Hash
		}

		entitlements = append(entitlements, entRef)
	}

	// Dispatch badge
	if badge, badgeEnt, hasBadge := cosmetics.ActiveBadge(); hasBadge {
		if ent, err := structures.ConvertEntitlement[structures.EntitlementDataBase](badgeEnt.ToRaw()); err == nil {
			dispatchCosmetic(badge.ToRaw(), ent)
		}
	}

	// Dispatch paint
	if paint, paintEnt, hasPaint := cosmetics.ActivePaint(); hasPaint {
		if ent, err := structures.ConvertEntitlement[structures.EntitlementDataBase](paintEnt.ToRaw()); err == nil {
			dispatchCosmetic(paint.ToRaw(), ent)
		}
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
				ID:     es.ID,
				Kind:   structures.ObjectKindEmoteSet,
				Object: utils.ToJSON(p.modelizer.EmoteSet(es)),
			}, eventCond)

			// Dispatch the Emote Set entitlement
			_ = p.events.Dispatch(ctx, events.EventTypeCreateEntitlement, events.ChangeMap{
				ID:         ent.ID,
				Kind:       structures.ObjectKindEntitlement,
				Contextual: true,
				Object:     utils.ToJSON(p.modelizer.Entitlement(ent.ToRaw(), user)),
			}, eventCond, entEventCond)

			// Add to presence's entitlement refs
			entitlements = append(entitlements, structures.UserPresenceEntitlement{
				Kind:  ent.Kind,
				ID:    ent.ID,
				RefID: ent.Data.RefID,
			})
		}
	}

	// Send delete events for any entitlements that are no longer active
	for _, entitlement := range presence.Entitlements {
		found := false

		for _, newEntitlement := range entitlements {
			if newEntitlement.ID == entitlement.ID {
				found = true
				break
			}
		}

		if !found {
			// Entitlement is no longer active, send delete event
			_ = p.events.Dispatch(ctx, events.EventTypeDeleteEntitlement, events.ChangeMap{
				ID:         entitlement.ID,
				Kind:       structures.ObjectKindEntitlement,
				Contextual: true,
			}, eventCond, entEventCond)
		}
	}

	// Update presence
	if _, err := p.mongo.Collection(mongo.CollectionNameUserPresences).UpdateOne(ctx, bson.M{
		"_id": presence.ID,
	}, bson.M{
		"$set": bson.M{
			"entitlements": entitlements,
		},
	}); err != nil {
		zap.S().Errorw("failed to update presence entitlements",
			"presence_id", presence.ID.Hex(),
			"user_id", presence.UserID.Hex(),
			"error", err.Error(),
		)
	}

	return nil
}
