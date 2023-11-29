package query

import (
	"context"
	"strings"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"

	"github.com/seventv/api/internal/search"
)

const EMOTES_QUERY_LIMIT = 300

func (q *Query) SearchEmotes(ctx context.Context, opt SearchEmotesOptions) ([]structures.Emote, int, error) {
	req := search.EmoteSearchOptions{
		Limit: int64(opt.Limit),
		Page:  int64(opt.Page) + 1,
		Sort: search.EmoteSortOptions{
			By:        "channel_count",
			Ascending: false,
		},
		Lifecycle: int32(structures.EmoteLifecycleLive),
	}

	// Define limit (how many emotes can be returned in a single query)
	if req.Limit > EMOTES_QUERY_LIMIT {
		req.Limit = EMOTES_QUERY_LIMIT
	} else if req.Limit < 1 {
		req.Limit = 1
	}

	// Define page
	if req.Page < 1 {
		req.Page = 1
	}

	// Define default filter
	//filter := opt.Filter
	//if filter == nil {
	//	filter = &SearchEmotesFilter{
	//		CaseSensitive: utils.PointerOf(false),
	//		ExactMatch:    utils.PointerOf(false),
	//		IgnoreTags:    utils.PointerOf(false),
	//		Document:      bson.M{},
	//	}
	//}

	// Define the query string
	query := strings.Trim(opt.Query, " ")

	// Set up db query

	// Apply permission checks
	// omit unlisted/private emotes
	if opt.Actor == nil || !opt.Actor.HasPermission(structures.RolePermissionEditAnyEmote) {
		req.Listed = true
	}

	if opt.Sort != nil && len(opt.Sort) > 0 {
		for key, value := range opt.Sort {
			sort := search.EmoteSortOptions{
				By: key,
			}
			if value.(int32) > 0 {
				sort.Ascending = true
			}
			req.Sort = sort
			break
		}
	}

	result, totalCount, err := q.search.SearchEmotes(query, req)
	zap.S().Infof("found emotes for query: \"%v\" \n%+v", query, result)

	emoteIds := bson.A{}

	for _, emote := range result {
		emoteIds = append(emoteIds, emote.Id)
	}

	emotes, err := q.Emotes(ctx, bson.M{"_id": bson.M{"$in": emoteIds}}).Items()
	if err != nil {
		zap.S().Errorw("mongo, failed to find emotes() gql query",
			"error", err,
		)

		return nil, 0, errors.ErrInternalServerError().SetDetail(err.Error())
	}

	return emotes, int(totalCount), nil
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
