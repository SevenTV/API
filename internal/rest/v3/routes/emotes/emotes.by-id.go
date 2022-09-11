package emotes

import (
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/middleware"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/common/errors"
)

type emoteRoute struct {
	Ctx global.Context
}

func newEmote(gctx global.Context) rest.Route {
	return &emoteRoute{gctx}
}

func (r *emoteRoute) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:      "/{emote.id}",
		Method:   rest.GET,
		Children: []rest.Route{},
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 300, []string{"s-maxage=600"}),
		},
	}
}

// @Summary Get Emote
// @Description Get emote by ID
// @Param emoteID path string true "ID of the emote"
// @Tags emotes
// @Produce json
// @Success 200 {object} model.EmoteModel
// @Router /emotes/{emote.id} [get]
func (r *emoteRoute) Handler(ctx *rest.Ctx) rest.APIError {
	emoteID, err := ctx.UserValue("emote.id").ObjectID()
	if err != nil {
		return errors.From(err)
	}

	emote, err := r.Ctx.Inst().Loaders.EmoteByID().Load(emoteID)
	if err != nil {
		return errors.From(err)
	}

	return ctx.JSON(rest.OK, r.Ctx.Inst().Modelizer.Emote(emote))
}
