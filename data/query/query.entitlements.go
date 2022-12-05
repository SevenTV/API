package query

import (
	"context"

	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
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

	if err := cur.All(ctx, &items); err != nil {
		zap.S().Errorw("failed to decode entitlements", "error", err)
		r.setError(err)

		return r
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

type QueryEntitlementsOptions struct {
	SelectedOnly bool
}
