package emotes

import (
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/middleware"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/api/internal/rest/v2/model"
	"github.com/seventv/common/errors"
	v2structures "github.com/seventv/common/structures/v2"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type globals struct {
	Ctx global.Context
}

func newGlobals(gCtx global.Context) rest.Route {
	return &globals{gCtx}
}

func (r *globals) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:      "/global",
		Method:   rest.GET,
		Children: []rest.Route{},
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 1800, nil),
		},
	}
}

// Get Global Emotes
// @Summary Get Globla Emotes
// @Description Lists active global emotes
// @Tags emotes
// @Produce json
// @Success 200 {array} model.Emote
// @Router /emotes/global [get]
func (r *globals) Handler(ctx *rest.Ctx) errors.APIError {
	es, err := r.Ctx.Inst().Query.GlobalEmoteSet(ctx)
	if err != nil {
		return errors.From(err)
	}

	result := make([]model.Emote, len(es.Emotes))

	emoteIDs := utils.Map(es.Emotes, func(a structures.ActiveEmote) primitive.ObjectID {
		return a.ID
	})

	emotes, _ := r.Ctx.Inst().Loaders.EmoteByID().LoadAll(emoteIDs)

	emoteMap := map[primitive.ObjectID]structures.Emote{}
	for _, emote := range emotes {
		emoteMap[emote.ID] = emote
	}

	for i, ae := range es.Emotes {
		e := utils.PointerOf(emoteMap[ae.ID])
		ae.Emote = e

		if ae.Emote == nil {
			continue
		}

		ae.Emote.Name = ae.Name

		result[i] = *model.NewEmote(*ae.Emote, r.Ctx.Config().CdnURL)
		result[i].Visibility |= v2structures.EmoteVisibilityGlobal
	}

	return ctx.JSON(rest.OK, result)
}
