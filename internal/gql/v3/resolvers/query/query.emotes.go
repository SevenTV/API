package query

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/Common/structures/v3/query"
	"github.com/SevenTV/Common/utils"
	"github.com/hashicorp/go-multierror"
	"github.com/meilisearch/meilisearch-go"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const EMOTES_QUERY_LIMIT = 300

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

type SearchHit struct {
	ID           primitive.ObjectID `json:"id"`
	Name         string             `json:"name"`
	Tags         []string           `json:"tags"`
	OwnerID      primitive.ObjectID `json:"owner_id"`
	Listed       bool               `json:"listed"`
	ChannelCount int                `json:"channel_count"`
	CreatedAt    int                `json:"created_at"`
}

func isTrue(b *bool) bool {
	return b != nil && *b
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
	sortopt := model.Sort{
		Value: "popularity",
		Order: model.SortOrderDescending,
	}
	if sortArg != nil {
		sortopt = *sortArg
	}

	var (
		result     []structures.Emote
		totalCount int
	)

	order, validOrder := sortOrderMap[sortopt.Order]
	field, validField := sortFieldMap[sortopt.Value]

	if r.Ctx.Inst().MeilieSearch.IsHealthy() && (!isTrue(filter.CaseSensitive) && !isTrue(filter.ExactMatch) && !isTrue(filter.IgnoreTags)) {
		sort := []string{}
		filters := []string{}

		var filter *string
		if actor == nil || !actor.HasPermission(structures.RolePermissionEditAnyEmote) {
			filters = append(filters, "listed = true")
		}
		if len(filters) != 0 {
			filter = utils.PointerOf("(" + strings.Join(filters, ") and (") + ")")
		}

		if validOrder && validField {
			field := "created_at"
			if sortopt.Value == "popularity" {
				field = "channel_count"
			}

			if sortopt.Order == model.SortOrderAscending {
				field += ":asc"
			} else {
				field += ":desc"
			}
			sort = append(sort, field)
		}

		resp, err := r.Ctx.Inst().MeilieSearch.Index("emotes").Search(queryValue, &meilisearch.SearchRequest{
			Offset: int64((page - 1) * limit),
			Limit:  int64(limit),
			Sort:   sort,
			Filter: filter,
		})
		if err != nil {
			return nil, err
		}

		if len(resp.Hits) != 0 {
			hits := []SearchHit{}
			rawHits, _ := json.Marshal(resp.Hits)
			err := json.Unmarshal(rawHits, &hits)
			if err != nil {
				return nil, err
			}

			ids := make([]primitive.ObjectID, len(hits))
			for i, hit := range hits {
				ids[i] = hit.ID
			}

			var errs []error
			result, errs = r.Ctx.Inst().Loaders.EmoteByID().LoadAll(ids)

			for _, v := range errs {
				err = multierror.Append(err, v).ErrorOrNil()
			}
			if err != nil {
				return nil, err
			}

			totalCount = int(resp.NbHits)
		}
	} else {
		// Define sorting
		// (will be ignored in the case of exact search)
		sortMap := bson.M{}
		if validField && validOrder {
			sortMap = bson.M{field: order}
		}

		// Run query
		var err error
		result, totalCount, err = r.Ctx.Inst().Query.SearchEmotes(ctx, query.SearchEmotesOptions{
			Actor: actor,
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
		if err != nil {
			return nil, err
		}
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

var sortFieldMap = map[string]string{
	"age":        "_id",
	"popularity": "versions.state.channel_count",
}
