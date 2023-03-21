package query

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/data/model/modelgql"
	"github.com/seventv/api/data/query"
	"github.com/seventv/api/internal/api/gql/v3/auth"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/api/internal/svc/limiter"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
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

	return modelgql.EmoteModel(r.Ctx.Inst().Modelizer.Emote(emote)), nil
}

func (r *Resolver) EmotesByID(ctx context.Context, list []primitive.ObjectID) ([]*model.EmotePartial, error) {
	emotes, errs := r.Ctx.Inst().Loaders.EmoteByID().LoadAll(list)
	if err := multierror.Append(nil, errs...).ErrorOrNil(); err != nil {
		r.Z().Errorw("failed to load emotes", "error", err)

		return nil, nil
	}

	result := make([]*model.EmotePartial, len(emotes))

	for i, emote := range emotes {
		result[i] = modelgql.EmotePartialModel(r.Ctx.Inst().Modelizer.Emote(emote).ToPartial())
	}

	return result, nil
}

func (r *Resolver) Emotes(ctx context.Context, queryValue string, pageArg *int, limitArg *int, filterArg *model.EmoteSearchFilter, sortArg *model.Sort) (*model.EmoteSearchResult, error) {
	// Rate limit
	if ok := r.Ctx.Inst().Limiter.Test(ctx, "search-emotes", 10, time.Second*5, limiter.TestOptions{
		Incr: 1,
	}); !ok {
		return nil, errors.ErrRateLimited()
	}

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

	if page > r.Ctx.Config().Limits.MaxPage {
		page = r.Ctx.Config().Limits.MaxPage
	}

	// Retrieve sorting options
	sortopt := &model.Sort{
		Value: "popularity",
		Order: model.SortOrderDescending,
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
	case model.EmoteSearchCategoryTrendingDay:
		ids, useMap, err2 := r.emoteCategoryTrending(ctx, trendingCategoryOptions{
			Days: map[model.EmoteSearchCategory]uint32{
				model.EmoteSearchCategoryTrendingDay: 1,
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
		extraDoc := bson.M{}

		// Flag fitlers
		flags := structures.BitField[structures.EmoteFlag](0)

		if filter.ZeroWidth != nil && *filter.ZeroWidth {
			flags = flags.Set(structures.EmoteFlagsZeroWidth)
		}

		if filter.Authentic != nil && *filter.Authentic {
			flags = flags.Set(structures.EmoteFlagsAuthentic)
		}

		if flags.Value() != 0 {
			extraDoc["flags"] = flags
		}

		// Aspect ratio filter
		if filter.AspectRatio != nil && *filter.AspectRatio != "" {
			sp := strings.Split(*filter.AspectRatio, ":")
			if len(sp) < 2 {
				return nil, errors.ErrInvalidRequest().SetDetail("Invalid format for aspect ratio")
			}

			// Parse the aspect ratio
			r1, er1 := strconv.ParseFloat(sp[0], 32)
			r2, er2 := strconv.ParseFloat(sp[1], 32)

			// Parse tolerance
			var tolerance uint8

			if len(sp) >= 3 {
				t, err := strconv.ParseUint(sp[2], 10, 8)
				if err != nil || t > 100 {
					return nil, errors.ErrInvalidRequest().SetDetail("Invalid format for aspect ratio (bad tolerance value)")
				}

				tolerance = uint8(t)
			}

			if er1 != nil || er2 != nil {
				return nil, errors.ErrInvalidRequest().SetDetail("Invalid format for aspect ratio (could not parse int)")
			}

			// Calculate the width / height values
			wMin := math.Floor(float64(128) * (r1))
			w := bson.M{
				"$gte": wMin * (1 - float64(tolerance)/100),
				"$lte": wMin * ((float64(tolerance) / 100) + 1),
			}

			hMin := math.Floor(float64(128) * (r2))
			h := bson.M{
				"$gte": hMin * (1 - float64(tolerance)/100),
				"$lte": hMin * ((float64(tolerance) / 100) + 1),
			}

			extraDoc["versions.image_files"] = bson.M{
				"$elemMatch": bson.M{
					"name":         "4x",
					"content_type": "image/webp",
					"width":        w,
					"height":       h,
				},
			}
		}

		// Animated
		if filter.Animated != nil && *filter.Animated {
			extraDoc["versions.animated"] = true
		}

		// Personal Use
		if filter.PersonalUse != nil {
			extraDoc["versions.state.allow_personal"] = *filter.PersonalUse
		}

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
				Document:      extraDoc,
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

		models[i] = modelgql.EmoteModel(r.Ctx.Inst().Modelizer.Emote(e))
	}

	return &model.EmoteSearchResult{
		Count:   totalCount,
		MaxPage: r.Ctx.Config().Limits.MaxPage,
		Items:   models,
	}, nil
}
