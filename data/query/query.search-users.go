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
)

func (q *Query) SearchUsers(ctx context.Context, filter bson.M, opts ...UserSearchOptions) ([]structures.User, error) {
	mtx := q.mtx("SearchUsers")
	mtx.Lock()
	defer mtx.Unlock()

	items := []structures.User{}

	paginate := mongo.Pipeline{}
	search := len(opts) > 0

	if search {
		opt := opts[0]
		sort := bson.M{"_id": -1}

		if len(opt.Sort) > 0 {
			sort = opt.Sort
		}

		paginate = append(paginate, []bson.D{
			{{Key: "$sort", Value: sort}},
			{{Key: "$limit", Value: opt.Limit}},
		}...)

		if opt.Query != "" {
			filter["$expr"] = bson.M{
				"$gt": bson.A{
					bson.M{"$indexOfCP": bson.A{
						"$username",
						strings.ToLower(opt.Query),
					}},
					-1,
				},
			}
		}
	}

	cur, err := q.mongo.Collection(mongo.CollectionNameUsers).Aggregate(ctx, aggregations.Combine(
		mongo.Pipeline{
			{{
				Key:   "$match",
				Value: filter,
			}},
			{{
				Key:   "$project",
				Value: bson.M{"_id": 1},
			}},
		},
		paginate,
	))
	if err != nil {
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
