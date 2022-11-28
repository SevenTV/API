package query

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/aggregations"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func (q *Query) SearchUsers(ctx context.Context, filter bson.M, opts ...UserSearchOptions) ([]structures.User, int, error) {
	mtx := q.mtx("SearchUsers")
	mtx.Lock()
	defer mtx.Unlock()

	items := []structures.User{}

	paginate := mongo.Pipeline{}
	search := len(opts) > 0 && opts[0].Page != 0
	if search {
		opt := opts[0]
		sort := bson.M{"_id": -1}
		if len(opt.Sort) > 0 {
			sort = opt.Sort
		}
		paginate = append(paginate, []bson.D{
			{{Key: "$sort", Value: sort}},
			{{Key: "$skip", Value: (opt.Page - 1) * opt.Limit}},
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

	b, _ := bson.Marshal(filter)
	h := sha256.New()
	h.Write(b)
	queryKey := q.redis.ComposeKey("common", fmt.Sprintf("user-search:%s", hex.EncodeToString(h.Sum(nil))))

	bans, err := q.Bans(ctx, BanQueryOptions{ // remove emotes made by usersa who own nothing and are happy
		Filter: bson.M{"effects": bson.M{"$bitsAnySet": structures.BanEffectMemoryHole}},
	})
	if err != nil {
		return nil, 0, err
	}

	cur, err := q.mongo.Collection(mongo.CollectionNameUsers).Aggregate(ctx, aggregations.Combine(
		mongo.Pipeline{
			{{
				Key:   "$match",
				Value: filter,
			}},
			{{
				Key: "$set",
				Value: bson.M{ // Remove memory holed editors
					"editors": bson.M{"$filter": bson.M{
						"input": "$editors",
						"as":    "e",
						"cond":  bson.M{"$not": bson.M{"$in": bson.A{"$$e.id", bans.MemoryHole.KeySlice()}}},
					}},
				},
			}},
		},
		paginate,
		mongo.Pipeline{
			{{
				Key: "$group",
				Value: bson.M{
					"_id": nil,
					"users": bson.M{
						"$push": "$$ROOT",
					},
				},
			}},
			{{
				Key: "$lookup",
				Value: mongo.Lookup{
					From:         mongo.CollectionNameEntitlements,
					LocalField:   "users._id",
					ForeignField: "user_id",
					As:           "role_entitlements",
				},
			}},
			{{
				Key: "$set",
				Value: bson.M{
					"role_entitlements": bson.M{
						"$filter": bson.M{
							"input": "$role_entitlements",
							"as":    "ent",
							"cond": bson.M{
								"$eq": bson.A{"$$ent.kind", structures.EntitlementKindRole},
							},
						},
					},
				},
			}},
		},
	))
	if err != nil {
		return items, 0, err
	}

	// Count the documents
	totalCount, countErr := q.redis.RawClient().Get(ctx, queryKey.String()).Int()
	if search && countErr == redis.Nil {
		cur, err := q.mongo.Collection(mongo.CollectionNameUsers).Aggregate(ctx, aggregations.Combine(
			mongo.Pipeline{
				{{Key: "$match", Value: filter}},
			},
			mongo.Pipeline{
				{{Key: "$count", Value: "count"}},
				{{Key: "$project", Value: bson.M{"count": "$count"}}},
			},
		))
		result := make(map[string]int, 1)
		if err == nil {
			if ok := cur.Next(ctx); ok {
				if err = cur.Decode(&result); err != nil {
					zap.S().Errorw("mongo, couldn't count users",
						"error", err,
					)
				}
			}
			_ = cur.Close(ctx)
		}
		totalCount = result["count"]
		_ = q.redis.SetEX(ctx, queryKey, totalCount, time.Hour)
	}

	// Get roles
	roles, _ := q.Roles(ctx, bson.M{})
	roleMap := make(map[primitive.ObjectID]structures.Role)
	for _, role := range roles {
		roleMap[role.ID] = role
	}

	// Map all objects
	if ok := cur.Next(ctx); !ok {
		return items, 0, nil // nothing found!
	}
	v := &aggregatedUsersResult{}
	if err = cur.Decode(v); err != nil {
		return items, 0, err
	}

	qb := &QueryBinder{ctx, q}
	userMap, err := qb.MapUsers(v.Users, v.RoleEntitlements...)
	if err != nil {
		return nil, 0, err
	}

	for _, u := range userMap {
		items = append(items, u)
	}

	return items, totalCount, multierror.Append(err, cur.Close(ctx)).ErrorOrNil()
}

type UserSearchOptions struct {
	Page  int
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
