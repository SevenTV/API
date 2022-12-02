package query

import (
	"context"
	"strconv"

	"github.com/seventv/api/data/query"
	"github.com/seventv/api/internal/gql/v2/gen/model"
	"github.com/seventv/api/internal/gql/v2/helpers"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
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

	// Global State
	filterDoc := bson.M{}

	exactVersion := make(utils.Set[primitive.ObjectID])
	modQueue := false

	namedMap := map[primitive.ObjectID]string{}

	if globalStateArg != nil && *globalStateArg != "include" {
		set, err := r.Ctx.Inst().Query.GlobalEmoteSet(ctx)
		if err == nil {
			ids := make([]primitive.ObjectID, len(set.Emotes))
			for i, ae := range set.Emotes {
				ids[i] = ae.ID
				namedMap[ae.ID] = ae.Name
			}

			switch *globalStateArg {
			case "only":
				filterDoc["versions.id"] = bson.M{"$in": ids}

				exactVersion.Fill(ids...)
			case "hide":
				filterDoc["versions.id"] = bson.M{"$not": bson.M{"$in": ids}}
			}
		}
	} else {
		return nil, errors.ErrInsufficientPrivilege().SetDetail("This endpoint is no longer available. Please use V3")
	}

	result, totalCount, err := r.Ctx.Inst().Query.SearchEmotes(ctx, query.SearchEmotesOptions{
		Actor: &actor,
		Limit: 50,
		Filter: &query.SearchEmotesFilter{
			Document: filterDoc,
		},
	})
	if err != nil {
		return nil, err
	}

	models := make([]*model.Emote, len(result))

	for i, e := range result {
		if len(e.Versions) > 0 {
			foundExact := false

			for _, ver := range e.Versions {
				if name, ok := namedMap[ver.ID]; ok {
					e.Name = name
				}

				if !exactVersion.Has(ver.ID) {
					continue
				}

				e.ID = ver.ID

				foundExact = true

				break
			}

			if !foundExact {
				// Bring forward the latest version
				ver := e.GetLatestVersion(!modQueue)
				if !ver.ID.IsZero() {
					e.ID = ver.ID
				}
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
