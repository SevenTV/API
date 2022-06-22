package query

import (
	"context"
	"strconv"

	"github.com/seventv/api/internal/gql/v2/gen/model"
	"github.com/seventv/api/internal/gql/v2/helpers"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/common/errors"
	v2structures "github.com/seventv/common/structures/v2"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/query"
	"github.com/seventv/common/utils"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) Emote(ctx context.Context, id string) (*model.Emote, error) {
	eid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.ErrBadObjectID()
	}

	emote, err := r.Ctx.Inst().Loaders.EmoteByID().Load(eid)
	if emote.ID.IsZero() || emote.ID == structures.DeletedEmote.ID {
		return nil, errors.ErrUnknownEmote()
	}

	return helpers.EmoteStructureToModel(emote, r.Ctx.Config().CdnURL), err
}

func (r *Resolver) SearchEmotes(
	ctx context.Context,
	queryArg string,
	limitArg *int,
	pageArg *int,
	pageSizeArg *int,
	submittedBy *string,
	globalStateArg *string,
	sortByArg *string,
	sortOrderArg *int,
	channel *string,
	filterArg *model.EmoteFilter,
) ([]*model.Emote, error) {
	actor := auth.For(ctx)

	// Define page
	page := 1
	if pageArg != nil && *pageArg > 1 {
		page = *pageArg
	}
	// Define limit
	// This is how many emotes can be searched in one request at most
	limit := 20
	if limitArg != nil {
		limit = *limitArg
	}

	if limit > query.EMOTES_QUERY_LIMIT {
		limit = query.EMOTES_QUERY_LIMIT
	}

	// Define sorting
	if sortByArg == nil {
		sortByArg = utils.PointerOf("popularity")
	}

	if sortOrderArg == nil {
		sortOrderArg = utils.PointerOf(0)
	}

	sortField, validField := sortFieldMap[*sortByArg]
	sortOrder, validOrder := sortOrderMap[*sortOrderArg]
	sortMap := bson.M{}

	if validField && validOrder {
		sortMap = bson.M{sortField: sortOrder}
	}

	// Global State
	filterDoc := bson.M{}
	onlyListed := true

	if globalStateArg != nil && *globalStateArg != "include" {
		set, err := r.Ctx.Inst().Query.GlobalEmoteSet(ctx)
		if err == nil {
			ids := make([]primitive.ObjectID, len(set.Emotes))
			for i, ae := range set.Emotes {
				ids[i] = ae.ID
			}

			switch *globalStateArg {
			case "only":
				filterDoc["versions.id"] = bson.M{"$in": ids}
			case "hide":
				filterDoc["versions.id"] = bson.M{"$not": bson.M{"$in": ids}}
			}
		}
	}

	if filterArg != nil {
		var (
			vis  int32
			visc int32
		)

		if filterArg.Visibility != nil {
			vis = int32(*filterArg.Visibility)
		}

		if filterArg.VisibilityClear != nil {
			visc = int32(*filterArg.VisibilityClear)
		}

		// Handle legacy mod queue
		// Where visibility: 4, visibility_clear: 256, meaning emotes pending approval
		if vis == v2structures.EmoteVisibilityUnlisted && visc == v2structures.EmoteVisibilityPermanentlyUnlisted {
			// Fetch mod items
			result, err := r.Ctx.Inst().Query.ModRequestMessages(ctx, query.ModRequestMessagesQueryOptions{
				Actor: actor,
				Targets: map[structures.ObjectKind]bool{
					structures.ObjectKindEmote: true,
				},
			}).Items()
			if err != nil {
				return nil, err
			}

			// Fetch emotes
			emoteIDs := make([]primitive.ObjectID, len(result))

			for i, msg := range result {
				if msg, err := structures.ConvertMessage[structures.MessageDataModRequest](msg); err == nil {
					emoteIDs[i] = msg.Data.TargetID
				}
			}
			// Set to filter
			filterDoc["versions.id"] = bson.M{
				"$in": emoteIDs,
			}
			onlyListed = false
		}
	}

	result, totalCount, err := r.Ctx.Inst().Query.SearchEmotes(ctx, query.SearchEmotesOptions{
		Actor: actor,
		Query: queryArg,
		Page:  page,
		Limit: limit,
		Sort:  sortMap,
		Filter: &query.SearchEmotesFilter{
			Document: filterDoc,
		},
	})
	if err != nil {
		return nil, err
	}

	models := make([]*model.Emote, len(result))

	for i, e := range result {
		// Bring forward the latest version
		if len(e.Versions) > 0 {
			ver := e.GetLatestVersion(onlyListed)
			if !ver.ID.IsZero() {
				e.ID = ver.ID
			}
		}

		models[i] = helpers.EmoteStructureToModel(e, r.Ctx.Config().CdnURL)
	}

	rctx, _ := ctx.Value(helpers.RequestCtxKey).(*fasthttp.RequestCtx)
	if rctx != nil {
		rctx.Response.Header.Set("X-Collection-Size", strconv.Itoa(totalCount))
	}

	return models, nil
}

var sortFieldMap = map[string]string{
	"age":        "_id",
	"popularity": "versions.state.channel_count",
}

var sortOrderMap = map[int]int{
	1: 1,
	0: -1,
}
