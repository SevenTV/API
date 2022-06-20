package user

import (
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/api/internal/rest/v2/model"
	"github.com/seventv/common/errors"
)

type emotes struct {
	Ctx global.Context
}

func newEmotes(gCtx global.Context) rest.Route {
	return &emotes{gCtx}
}

func (*emotes) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:        "/emotes",
		Method:     rest.GET,
		Children:   []rest.Route{},
		Middleware: []rest.Middleware{},
	}
}

// Get Channel Emotes
// @Summary Get Channel Emotes
// @Description List the channel emotes of a user
// @Tags users,emotes
// @Param user path string false "User ID, Twitch ID or Twitch Login"
// @Produce json
// @Success 200 {array} model.Emote
// @Router /users/{user}/emotes [get]
func (r *emotes) Handler(ctx *rest.Ctx) errors.APIError {
	key, _ := ctx.UserValue("user").String()
	user, err := r.Ctx.Inst().Loaders.UserByUsername().Load(key)
	if err != nil {
		return errors.From(err)
	}
	if user.ID.IsZero() {
		return errors.ErrUnknownUser()
	}

	// Fetch user's channel emoes
	con, _, err := user.Connections.Twitch()
	if err != nil {
		return errors.ErrUnknownUser().SetDetail("No Twitch Connection but this is a v2 request")
	}

	emoteSet, err := r.Ctx.Inst().Loaders.EmoteSetByID().Load(con.EmoteSetID)
	if err != nil {
		return errors.From(err)
	}

	result := []*model.Emote{}
	for _, e := range emoteSet.Emotes {
		if e.Emote != nil {
			result = append(result, model.NewEmote(*e.Emote, r.Ctx.Config().CdnURL))
		}
	}

	return ctx.JSON(rest.OK, result)
}
