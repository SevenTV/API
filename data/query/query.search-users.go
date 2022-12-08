package query

import (
	"context"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/aggregations"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func (q *Query) SearchUsers(ctx context.Context, filter bson.M, opts ...UserSearchOptions) ([]structures.User, error) {
	mtx := q.mtx("SearchUsers")
	mtx.Lock()
	defer mtx.Unlock()

	items := []structures.User{}

	paginate := mongo.Pipeline{}

	if len(opts) > 0 {
		opt := opts[0]
		sort := bson.M{"searchIndex": 1, "exact": -1}

		for k, v := range opt.Sort {
			sort[k] = v
		}

		paginate = append(paginate, []bson.D{
			{{
				Key: "$set",
				Value: bson.M{
					"searchIndex": bson.M{"$indexOfCP": bson.A{
						"$username",
						strings.ToLower(opt.Query),
					}},
					"exact": bson.M{"$cond": bson.M{
						"if":   bson.M{"$eq": bson.A{"$username", strings.ToLower(opt.Query)}},
						"then": true,
						"else": false,
					}},
				},
			}},
			{{Key: "$match", Value: bson.M{
				"searchIndex": bson.M{"$gte": 0},
			}}},
			{{Key: "$sort", Value: sort}},
			{{Key: "$limit", Value: opt.Limit}},
		}...)
	}

	cur, err := q.mongo.Collection(mongo.CollectionNameUsers).Aggregate(ctx, aggregations.Combine(
		mongo.Pipeline{
			{{
				Key:   "$match",
				Value: filter,
			}},
		},
		paginate,
	))
	if err != nil {
		zap.S().Errorw("failed to aggregate search users", "error", err)

		return items, err
	}

	// Map all objects
	if err := cur.All(ctx, &items); err != nil {
		return items, nil // nothing found!
	}

	return items, multierror.Append(err, cur.Close(ctx)).ErrorOrNil()
}

type UserSearchOptions struct {
	Limit int
	Query string
	Sort  bson.M
}
type aggregatedUsersResult struct {
	Users            []structures.User                  `bson:"users"`
	RoleEntitlements []structures.Entitlement[bson.Raw] `bson:"role_entitlements"`
	TotalCount       int                                `bson:"total_count"`
}

func (q *Query) UserEditorOf(ctx context.Context, id primitive.ObjectID) ([]structures.UserEditor, error) {
	cur, err := q.mongo.Collection(mongo.CollectionNameUsers).Aggregate(ctx, mongo.Pipeline{
		{{
			Key: "$match",
			Value: bson.M{
				"editors.id": id,
			},
		}},
		{{
			Key: "$project",
			Value: bson.M{
				"editor": bson.M{
					"$mergeObjects": bson.A{
						bson.M{"$first": bson.M{"$filter": bson.M{
							"input": "$editors",
							"as":    "ed",
							"cond": bson.M{
								"$eq": bson.A{"$$ed.id", id},
							},
						}}},
						bson.M{"id": "$_id"},
					},
				},
			},
		}},
		{{Key: "$replaceRoot", Value: bson.M{"newRoot": "$editor"}}},
	})
	if err != nil {
		return nil, err
	}

	v := []structures.UserEditor{}
	if err = cur.All(ctx, &v); err != nil {
		return nil, err
	}

	return v, nil
}
