package query

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/aggregations"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

const EMOTES_QUERY_LIMIT = 300

func (q *Query) SearchEmotes(ctx context.Context, opt SearchEmotesOptions) ([]structures.Emote, int, error) {
	// Define limit (how many emotes can be returned in a single query)
	limit := opt.Limit
	if limit > EMOTES_QUERY_LIMIT {
		limit = EMOTES_QUERY_LIMIT
	} else if limit < 1 {
		limit = 1
	}

	// Define page
	page := 1
	if opt.Page > page {
		page = opt.Page
	} else if opt.Page < 1 {
		page = 1
	}

	// Define default filter
	filter := opt.Filter
	if filter == nil {
		filter = &SearchEmotesFilter{
			CaseSensitive: utils.PointerOf(false),
			ExactMatch:    utils.PointerOf(false),
			IgnoreTags:    utils.PointerOf(false),
			Document:      bson.M{},
		}
	}

	// Define the query string
	query := strings.Trim(opt.Query, " ")

	// Set up db query
	bans, err := q.Bans(ctx, BanQueryOptions{ // remove emotes made by usersa who own nothing and are happy
		Filter: bson.M{"effects": bson.M{"$bitsAnySet": structures.BanEffectNoOwnership | structures.BanEffectMemoryHole}},
	})
	if err != nil {
		return nil, 0, err
	}

	match := bson.D{
		{Key: "versions.state.lifecycle", Value: structures.EmoteLifecycleLive},
		{Key: "owner_id", Value: bson.M{"$not": bson.M{
			"$in": bans.NoOwnership.KeySlice(),
		}}},
	}
	if len(filter.Document) > 0 {
		for k, v := range filter.Document {
			match = append(match, bson.E{Key: k, Value: v})
		}
	}

	// Apply permission checks
	// omit unlisted/private emotes
	privileged := int(1)
	if opt.Actor == nil || !opt.Actor.HasPermission(structures.RolePermissionEditAnyEmote) {
		privileged = 0
		match = append(match, bson.E{
			Key:   "versions.state.listed",
			Value: true,
		})
	}

	// Define the pipeline
	pipeline := mongo.Pipeline{}

	// Apply name/tag query
	h := sha256.New()
	h.Write(utils.S2B(query))
	h.Write([]byte{byte(privileged)})
	if len(filter.Document) > 0 {
		optBytes, _ := json.Marshal(filter.Document)
		h.Write(optBytes)
	}

	queryKey := q.redis.ComposeKey("common", fmt.Sprintf("emote-search:%s", hex.EncodeToString((h.Sum(nil)))))
	cpargs := bson.A{}

	// Handle exact match
	if filter.ExactMatch != nil && *filter.ExactMatch {
		// For an exact mathc we will use the $text operator
		// rather than $indexOfCP because name/tags are indexed fields
		match = append(match, bson.E{Key: "$text", Value: bson.M{
			"$search":        query,
			"$caseSensitive": filter.CaseSensitive != nil && *filter.CaseSensitive,
		}})
		pipeline = append(pipeline, []bson.D{
			{{Key: "$match", Value: match}},
			{{Key: "$sort", Value: bson.M{"score": bson.M{"$meta": "textScore"}}}},
		}...)

		if opt.Sort != nil && len(opt.Sort) > 0 {
			pipeline = append(pipeline, bson.D{
				{Key: "$sort", Value: opt.Sort},
			})
		}
	} else {
		or := bson.A{}
		if filter.CaseSensitive != nil && *filter.CaseSensitive {
			cpargs = append(cpargs, "$name", query)
		} else {
			cpargs = append(cpargs, bson.M{"$toLower": "$name"}, strings.ToLower(query))
		}

		or = append(or, bson.M{
			"$expr": bson.M{
				"$gt": bson.A{bson.M{"$indexOfCP": cpargs}, -1},
			},
		})

		// Add tag search
		if filter.IgnoreTags == nil || !*filter.IgnoreTags {
			qVal := query
			if len(qVal) > 0 && qVal[0] == '#' {
				qVal = qVal[1:]
			}

			or = append(or, bson.M{
				"$expr": bson.M{
					"$gt": bson.A{
						bson.M{"$indexOfCP": bson.A{bson.M{"$reduce": bson.M{
							"input":        "$tags",
							"initialValue": " ",
							"in":           bson.M{"$concat": bson.A{"$$value", "$$this"}},
						}}, strings.ToLower(qVal)}},
						-1,
					},
				},
			})
		}

		match = append(match, bson.E{Key: "$or", Value: or})
		if opt.Sort != nil && len(opt.Sort) > 0 {
			pipeline = append(pipeline, bson.D{
				{Key: "$sort", Value: opt.Sort},
			})
		}
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: match}})
	}

	mtx := q.mtx("SearchEmotes")
	mtx.Lock()

	totalCount, countErr := q.redis.RawClient().Get(ctx, string(queryKey)).Int()
	wg := sync.WaitGroup{}

	if countErr == redis.Nil {
		wg.Add(1)
		go func() { // Run a separate pipeline to return the total count that could be paginated
			defer func() {
				mtx.Unlock()
				wg.Done()
			}()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()

			cur, err := q.mongo.Collection(mongo.CollectionNameEmotes).Aggregate(ctx, aggregations.Combine(
				pipeline,
				mongo.Pipeline{
					{{Key: "$count", Value: "count"}},
				}),
			)
			result := make(map[string]int, 1)
			if err == nil {
				cur.Next(ctx)
				if err = cur.Decode(&result); err != nil {
					if err != io.EOF {
						zap.S().Errorw("mongo, couldn't count",
							"error", err,
						)
					}
				}
				_ = cur.Close(ctx)
			}

			// Return total count & cache
			totalCount = result["count"]
			dur := utils.Ternary(query == "", time.Hour*4, time.Hour*2)
			if err = q.redis.SetEX(ctx, queryKey, totalCount, dur); err != nil {
				zap.S().Errorw("redis, failed to save total list count of emotes() gql query",
					"error", err,
					"key", queryKey,
					"count", totalCount,
				)
			}
		}()
	} else {
		mtx.Unlock()
	}

	// Paginate and fetch the relevant emotes
	result := []structures.Emote{}
	cur, err := q.mongo.Collection(mongo.CollectionNameEmotes).Aggregate(ctx, aggregations.Combine(
		pipeline,
		mongo.Pipeline{
			{{Key: "$skip", Value: (page - 1) * limit}},
			{{Key: "$limit", Value: limit}},
			{{
				Key: "$group",
				Value: bson.M{
					"_id": nil,
					"emotes": bson.M{
						"$push": "$$ROOT",
					},
				},
			}},
			{{
				Key: "$lookup",
				Value: mongo.Lookup{
					From:         mongo.CollectionNameUsers,
					LocalField:   "emotes.owner_id",
					ForeignField: "_id",
					As:           "emote_owners",
				},
			}},
			{{
				Key: "$lookup",
				Value: mongo.Lookup{
					From:         mongo.CollectionNameEntitlements,
					LocalField:   "emotes.owner_id",
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
		return nil, 0, errors.ErrInternalServerError().SetDetail(err.Error())
	}

	v := &aggregatedEmotesResult{}
	cur.Next(ctx)
	if err = cur.Decode(v); err != nil {
		if err == io.EOF {
			return nil, 0, errors.ErrNoItems()
		}

		return nil, 0, err
	}

	// Map all objects
	qb := &QueryBinder{ctx, q}
	ownerMap, err := qb.MapUsers(v.EmoteOwners, v.RoleEntitlements...)
	if err != nil {
		return nil, 0, err
	}

	for _, e := range v.Emotes { // iterate over emotes
		if e.ID.IsZero() {
			continue
		}

		if _, banned := bans.MemoryHole[e.OwnerID]; banned {
			e.OwnerID = primitive.NilObjectID
		} else {
			owner := ownerMap[e.OwnerID]
			e.Owner = &owner
		}

		result = append(result, e)
	}

	// wait for total count to finish
	wg.Wait()

	return result, totalCount, nil
}

type SearchEmotesOptions struct {
	Query  string
	Page   int
	Limit  int
	Filter *SearchEmotesFilter
	Sort   bson.M
	Actor  *structures.User
}

type SearchEmotesFilter struct {
	CaseSensitive *bool  `json:"cs"`
	ExactMatch    *bool  `json:"exm"`
	IgnoreTags    *bool  `json:"ignt"`
	Document      bson.M `json:"doc"`
}
