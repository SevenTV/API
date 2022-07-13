package emotes

import (
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/api/internal/rest/v2/model"
	"github.com/seventv/api/internal/rest/v3/middleware"
	"github.com/seventv/common/errors"
)

type emote struct {
	Ctx global.Context
}

func newEmote(gCtx global.Context) rest.Route {
	return &emote{gCtx}
}

func (r *emote) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:      "/{emote}",
		Method:   rest.GET,
		Children: []rest.Route{},
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 600, []string{"s-maxage=600"}),
		},
	}
}

// Get Emote
// @Summary Get Emote
// @Description Find an emote by its ID
// @Tags emotes
// @Param emote path string false "Emote ID"
// @Produce json
// @Success 200 {object} model.Emote
// @Router /emotes/{emote} [get]
func (r *emote) Handler(ctx *rest.Ctx) errors.APIError {
	emoteID, err := ctx.UserValue(rest.Key("emote")).ObjectID()
	if err != nil {
		return errors.From(err)
	}

	emote, err := r.Ctx.Inst().Loaders.EmoteByID().Load(emoteID)
	if err != nil {
		return errors.From(err)
	}

	if emote.ID.IsZero() {
		return errors.ErrUnknownEmote()
	}

	return ctx.JSON(rest.OK, model.NewEmote(emote, r.Ctx.Config().CdnURL))
}
