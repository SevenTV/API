package user

import (
	"strings"

	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/api/internal/rest/v2/model"
	"github.com/seventv/api/internal/rest/v3/middleware"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type emotes struct {
	Ctx global.Context
}

func newEmotes(gCtx global.Context) rest.Route {
	return &emotes{gCtx}
}

func (r *emotes) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:      "/emotes",
		Method:   rest.GET,
		Children: []rest.Route{},
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 60, []string{"s-maxage=60"}),
		},
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

	var id primitive.ObjectID
	if primitive.IsValidObjectID(key) {
		id, _ = primitive.ObjectIDFromHex(key)
	}

	filter := utils.Ternary(id.IsZero(), bson.M{"$or": bson.A{
		bson.M{"connections.id": key},
		bson.M{"username": strings.ToLower(key)},
	}}, bson.M{
		"_id": id,
	})

	user, err := r.Ctx.Inst().Query.Users(ctx, filter).First()
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
		if e.Emote == nil {
			continue
		}

		v := e.Emote.GetLatestVersion(false)
		if v.ID.IsZero() || v.IsUnavailable() || v.IsProcessing() {
			continue
		}

		e.Emote.Name = e.Name
		result = append(result, model.NewEmote(*e.Emote, r.Ctx.Config().CdnURL))
	}

	return ctx.JSON(rest.OK, result)
}
