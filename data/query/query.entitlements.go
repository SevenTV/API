package query

import (
	"context"

	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func (q *Query) Entitlements(ctx context.Context, filter bson.M, opts ...QueryEntitlementsOptions) *QueryResult[EntitlementQueryResult] {
	items := []EntitlementQueryResult{}
	r := &QueryResult[EntitlementQueryResult]{}

	opt := QueryEntitlementsOptions{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	// typeFilterFactory creates a condition map that filters for an entitlement type
	typeFilterFactory := func(kind structures.EntitlementKind, coll mongo.CollectionName, selectable bool) bson.M {
		a := bson.A{
			bson.M{"$eq": bson.A{"$$e.kind", kind}},
		}

		if selectable {
			a = append(a, bson.M{"$eq": bson.A{"$$e.data.selected", true}})
		}

		return bson.M{
			"$filter": bson.M{
				"input": "$ent",
				"as":    "e",
				"cond":  bson.M{"$and": a},
			},
		}
	}

	// Find entitlements, grouped by user, output categorized by kind
	cur, err := q.mongo.Collection(mongo.CollectionNameEntitlements).Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{
			Key: "$group",
			Value: bson.M{
				"_id": "$user_id",
				"ent": bson.M{
					"$push": "$$ROOT",
				},
			},
		}},
		// group the entitlements by type
		{{
			Key: "$project",
			Value: bson.M{
				"roles":      typeFilterFactory(structures.EntitlementKindRole, mongo.CollectionNameRoles, false),
				"badges":     typeFilterFactory(structures.EntitlementKindBadge, mongo.CollectionNameCosmetics, opt.SelectedOnly),
				"paints":     typeFilterFactory(structures.EntitlementKindPaint, mongo.CollectionNameCosmetics, opt.SelectedOnly),
				"emote_sets": typeFilterFactory(structures.EntitlementKindEmoteSet, mongo.CollectionNameEmoteSets, false),
			},
		}},
	})
	if err != nil {
		zap.S().Errorw("failed to query entitlements", "error", err)
		r.setError(err)

		return r
	}

	defer cur.Close(ctx)

	cosmeticIDs := make(utils.Set[primitive.ObjectID])

	for cur.Next(ctx) {
		var item EntitlementQueryResult

		if err := cur.Decode(&item); err != nil {
			zap.S().Errorw("failed to decode entitlements", "error", err)
			r.setError(err)

			return r
		}

		for _, b := range item.Badges {
			cosmeticIDs.Add(b.Data.RefID)
		}

		for _, p := range item.Paints {
			cosmeticIDs.Add(p.Data.RefID)
		}

		items = append(items, item)
	}

	if err := cur.All(ctx, &items); err != nil {
		zap.S().Errorw("failed to decode entitlements", "error", err)
		r.setError(err)

		return r
	}

	// Fetch references
	cosmetics, err := q.Cosmetics(ctx, cosmeticIDs)
	if err != nil {
		zap.S().Errorw("failed to fetch cosmetics", "error", err)
		r.setError(err)

		return r
	}

	cosmeticMap := make(map[primitive.ObjectID]structures.Cosmetic[bson.Raw])
	for _, c := range cosmetics {
		cosmeticMap[c.ID] = c
	}

	// Attach references
	for i := range items {
		roleIDs := make(utils.Set[primitive.ObjectID])
		for _, r := range items[i].Roles {
			roleIDs.Add(r.Data.RefID)
		}

		for j, e := range items[i].Badges {
			x, _ := structures.ConvertCosmetic[structures.CosmeticDataBadge](cosmeticMap[items[i].Badges[j].Data.RefID])

			if !e.Condition.IsMet(roleIDs) {
				continue
			}

			items[i].Badges[j].Data.RefObject = &x
		}

		for j, e := range items[i].Paints {
			x, _ := structures.ConvertCosmetic[structures.CosmeticDataPaint](cosmeticMap[items[i].Paints[j].Data.RefID])

			if !e.Condition.IsMet(roleIDs) {
				continue
			}

			items[i].Paints[j].Data.RefObject = &x
		}

		items[i].EmoteSets = utils.Filter(items[i].EmoteSets, func(x structures.Entitlement[structures.EntitlementDataEmoteSet]) bool {
			return x.Condition.IsMet(roleIDs)
		})
	}

	r.setItems(items)

	return r
}

type EntitlementQueryResult struct {
	UserID    primitive.ObjectID                                               `bson:"_id"`
	Roles     EntitlementQueryResultBucket[structures.EntitlementDataRole]     `bson:"roles"`
	Badges    EntitlementQueryResultBucket[structures.EntitlementDataBadge]    `bson:"badges"`
	Paints    EntitlementQueryResultBucket[structures.EntitlementDataPaint]    `bson:"paints"`
	EmoteSets EntitlementQueryResultBucket[structures.EntitlementDataEmoteSet] `bson:"emote_sets"`
}

type EntitlementQueryResultBucket[T structures.EntitlementData] []structures.Entitlement[T]

func (eqr *EntitlementQueryResult) ActivePaint() (structures.Cosmetic[structures.CosmeticDataPaint], structures.Entitlement[structures.EntitlementDataPaint], bool) {
	var (
		item structures.Cosmetic[structures.CosmeticDataPaint]
		ent  structures.Entitlement[structures.EntitlementDataPaint]
	)

	for _, p := range eqr.Paints {
		if p.Data.RefObject == nil {
			continue
		}

		if p.Data.Selected && (!item.ID.IsZero() && p.Data.RefObject.Priority < item.Priority) {
			continue
		}

		item = *p.Data.RefObject
		ent = p
	}

	return item, ent, !item.ID.IsZero()
}

func (eqr *EntitlementQueryResult) ActiveBadge() (structures.Cosmetic[structures.CosmeticDataBadge], structures.Entitlement[structures.EntitlementDataBadge], bool) {
	var (
		item structures.Cosmetic[structures.CosmeticDataBadge]
		ent  structures.Entitlement[structures.EntitlementDataBadge]
	)

	for _, b := range eqr.Badges {
		if b.Data.RefObject == nil {
			continue
		}

		if b.Data.Selected && (!item.ID.IsZero() && b.Data.RefObject.Priority < item.Priority) {
			continue
		}

		item = *b.Data.RefObject
		ent = b
	}

	return item, ent, !item.ID.IsZero()
}

type QueryEntitlementsOptions struct {
	SelectedOnly bool
}
