package query

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	_, err := q.Bans(ctx, BanQueryOptions{ // remove emotes made by usersa who own nothing and are happy
		Filter: bson.M{"effects": bson.M{"$bitsAnySet": structures.BanEffectNoOwnership | structures.BanEffectMemoryHole}},
	})
	if err != nil {
		return nil, 0, err
	}

	stateMatch := bson.D{
		{Key: "state.lifecycle", Value: structures.EmoteLifecycleLive},
	}

	// Apply permission checks
	// omit unlisted/private emotes
	privileged := int(1)
	if opt.Actor == nil || !opt.Actor.HasPermission(structures.RolePermissionEditAnyEmote) {
		privileged = 0

		stateMatch = append(stateMatch, bson.E{
			Key:   "state.listed",
			Value: true,
		})
	}

	match := bson.D{
		{Key: "versions", Value: bson.M{
			"$elemMatch": stateMatch,
		}},
	}

	if len(filter.Document) > 0 {
		for k, v := range filter.Document {
			match = append(match, bson.E{Key: k, Value: v})
		}
	}

	// Apply name/tag query
	h := sha256.New()
	h.Write(utils.S2B(query))
	h.Write([]byte{byte(privileged)})

	queryKey := q.redis.ComposeKey("common", fmt.Sprintf("emote-search:%s", hex.EncodeToString((h.Sum(nil)))))
	cpargs := bson.A{}

	sorter := bson.M{}

	// Handle exact match
	if filter.ExactMatch != nil && *filter.ExactMatch {
		// For an exact mathc we will use the $text operator
		// rather than $indexOfCP because name/tags are indexed fields
		match = append(match, bson.E{Key: "$text", Value: bson.M{
			"$search":        query,
			"$caseSensitive": filter.CaseSensitive != nil && *filter.CaseSensitive,
		}})

		sorter = bson.M{"score": bson.M{"$meta": "textScore"}}

		h.Write(utils.S2B("FILTER_EXACT"))
	}

	if len(query) > 0 {
		or := bson.A{}

		if filter.CaseSensitive != nil && *filter.CaseSensitive {
			cpargs = append(cpargs, "$name", query)

			h.Write(utils.S2B("FILTER_CASE_SENSITIVE"))
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

			h.Write(utils.S2B("FILTER_IGNORE_TAGS"))
		}

		if len(or) > 0 {
			match = append(match, bson.E{Key: "$or", Value: or})
		}
	}

	if opt.Sort != nil && len(opt.Sort) > 0 {
		sorter = opt.Sort
	}

	if len(filter.Document) > 0 {
		optBytes, _ := json.Marshal(filter.Document)
		h.Write(optBytes)
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

			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			value, err := q.mongo.Collection(mongo.CollectionNameEmotes).CountDocuments(ctx, match)
			if err != nil {
				zap.S().Errorw("mongo, failed to count emotes() gql query",
					"error", err,
					"match", match,
				)

				return
			}

			totalCount = int(value)

			// Return total count & cache
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

	wg.Wait()

	// Paginate and fetch the relevant emotes
	result := []structures.Emote{}
	cur, err := q.mongo.Collection(mongo.CollectionNameEmotes).Find(
		ctx,
		match,
		options.Find().
			SetSort(sorter).
			SetSkip(int64((page-1)*limit)).
			SetLimit(int64(limit)),
	)

	if err != nil {
		zap.S().Errorw("mongo, failed to find emotes() gql query",
			"error", err,
			"match", match,
			"sort", sorter,
			"skip", (page-1)*limit,
			"limit", limit,
		)

		return nil, 0, errors.ErrInternalServerError().SetDetail(err.Error())
	}

	if err = cur.All(ctx, &result); err != nil {
		return nil, 0, errors.ErrInternalServerError().SetDetail(err.Error())
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
