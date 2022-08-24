package query

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/query"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const EMOTES_QUERY_LIMIT = 300

var sortFieldMap = map[string]string{
	"age":        "_id",
	"popularity": "versions.state.channel_count",
}

func (r *Resolver) Emote(ctx context.Context, id primitive.ObjectID) (*model.Emote, error) {
	emote, err := r.Ctx.Inst().Loaders.EmoteByID().Load(id)
	if err != nil {
		return nil, err
	}

	if emote.ID.IsZero() || emote.ID == structures.DeletedEmote.ID {
		return nil, errors.ErrUnknownEmote()
	}

	return helpers.EmoteStructureToModel(emote, r.Ctx.Config().CdnURL), nil
}

func (r *Resolver) EmotesByID(ctx context.Context, list []primitive.ObjectID) ([]*model.EmotePartial, error) {
	emotes, errs := r.Ctx.Inst().Loaders.EmoteByID().LoadAll(list)
	if err := multierror.Append(nil, errs...).ErrorOrNil(); err != nil {
		r.Z().Errorw("failed to load emotes", "error", err)

		return nil, nil
	}

	result := make([]*model.EmotePartial, len(emotes))

	for i, emote := range emotes {
		result[i] = helpers.EmoteStructureToPartialModel(helpers.EmoteStructureToModel(emote, r.Ctx.Config().CdnURL))
	}

	return result, nil
}

func (r *Resolver) Emotes(ctx context.Context, queryValue string, pageArg *int, limitArg *int, filterArg *model.EmoteSearchFilter, sortArg *model.Sort) (*model.EmoteSearchResult, error) {
	actor := auth.For(ctx)

	// Define limit (how many emotes can be returned in a single query)
	limit := 20
	if limitArg != nil {
		limit = *limitArg
	}

	if limit > EMOTES_QUERY_LIMIT {
		limit = EMOTES_QUERY_LIMIT
	}

	// Define default filter
	filter := filterArg
	if filter == nil {
		filter = &model.EmoteSearchFilter{
			CaseSensitive: utils.PointerOf(false),
			ExactMatch:    utils.PointerOf(false),
		}
	} else {
		filter = filterArg
	}

	// Define the query string
	queryValue = strings.Trim(queryValue, " ")

	// Retrieve pagination values
	page := 1
	if pageArg != nil {
		page = *pageArg
	}

	if page < 1 {
		page = 1
	}

	// Retrieve sorting options
	sortopt := &model.Sort{
		Value: "popularity",
		Order: model.SortOrderAscending,
	}
	if sortArg != nil {
		sortopt = sortArg
	}

	// Define sorting
	// (will be ignored in the case of exact search)
	order, validOrder := sortOrderMap[string(sortopt.Order)]
	field, validField := sortFieldMap[sortopt.Value]
	sortMap := bson.M{}

	if validField && validOrder {
		sortMap = bson.M{field: order}
	}

	// Run query
	var (
		result     []structures.Emote
		totalCount int
		err        error
	)

	cat := model.EmoteSearchCategoryTop
	if filter.Category != nil {
		cat = *filter.Category
	}

	switch cat {
	case model.EmoteSearchCategoryTrendingDay, model.EmoteSearchCategoryTrendingWeek, model.EmoteSearchCategoryTrendingMonth:
		ids, useMap, err2 := r.emoteCategoryTrending(ctx, trendingCategoryOptions{
			Days: map[model.EmoteSearchCategory]uint32{
				model.EmoteSearchCategoryTrendingDay:   1,
				model.EmoteSearchCategoryTrendingWeek:  7,
				model.EmoteSearchCategoryTrendingMonth: 30,
			}[cat],
			UserMinAge:    7,
			EmoteMaxAge:   365,
			UsageThresold: 10,
			Limit:         250,
		})
		if err2 != nil {
			return nil, err
		}

		sort.Slice(ids, func(i, j int) bool {
			return useMap[ids[i]] > useMap[ids[j]]
		})

		totalCount = len(ids)

		// shrink the fetch list if needed
		if page > 1 {
			min := math.Min(float64(len(ids)), float64((page-1)*limit))
			ids = ids[int(min):]
		}

		if len(ids) > limit+1 {
			ids = ids[:limit]
		}

		emotes, errs := r.Ctx.Inst().Loaders.EmoteByID().LoadAll(ids)
		if err := multierror.Append(nil, errs...).ErrorOrNil(); err != nil {
			return nil, errors.ErrNoItems()
		}

		result = emotes
	default:
		result, totalCount, err = r.Ctx.Inst().Query.SearchEmotes(ctx, query.SearchEmotesOptions{
			Actor: &actor,
			Query: queryValue,
			Page:  page,
			Limit: limit,
			Sort:  sortMap,
			Filter: &query.SearchEmotesFilter{
				CaseSensitive: filter.CaseSensitive,
				ExactMatch:    filter.ExactMatch,
				IgnoreTags:    filter.IgnoreTags,
			},
		})
	}

	if err != nil {
		return nil, err
	}

	models := make([]*model.Emote, len(result))

	for i, e := range result {
		// Bring forward the latest version
		if len(e.Versions) > 0 {
			if ver := e.GetLatestVersion(true); !ver.ID.IsZero() {
				e.ID = ver.ID
			}
		}

		models[i] = helpers.EmoteStructureToModel(e, r.Ctx.Config().CdnURL)
	}

	return &model.EmoteSearchResult{
		Count: totalCount,
		Items: models,
	}, nil
}
