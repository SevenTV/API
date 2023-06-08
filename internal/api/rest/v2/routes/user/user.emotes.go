package user

import (
	"strings"

	"github.com/seventv/api/data/query"
	"github.com/seventv/api/internal/api/rest/middleware"
	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/api/rest/v2/model"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
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
			middleware.SetCacheControl(r.Ctx, 60, []string{"public"}),
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

	// Check ban
	bans, err := r.Ctx.Inst().Query.Bans(ctx, query.BanQueryOptions{
		Filter: bson.M{"victim_id": user.ID},
	})
	if err == nil && bans.MemoryHole.Has(user.ID) {
		return errors.ErrUnknownUser()
	}

	// Fetch user's channel emoes
	var con structures.UserConnection[bson.Raw]

	for _, c := range user.Connections {
		if key != c.ID {
			continue
		}

		con = c
	}

	if con.ID == "" {
		// try username
		tw, _, _ := user.Connections.Twitch()
		if tw.ID == "" && strings.ToLower(key) != tw.Data.Login {
			return errors.ErrUnknownUser()
		}

		con = tw.ToRaw()
	}

	set, err := r.Ctx.Inst().Loaders.EmoteSetByID().Load(con.EmoteSetID)
	if err != nil {
		return errors.From(err)
	}

	// Set relations
	emoteIDs := utils.Map(set.Emotes, func(a structures.ActiveEmote) primitive.ObjectID {
		return a.ID
	})

	emotes, _ := r.Ctx.Inst().Loaders.EmoteByID().LoadAll(emoteIDs)

	emoteMap := map[primitive.ObjectID]structures.Emote{}
	for _, emote := range emotes {
		emoteMap[emote.ID] = emote
	}

	result := []*model.Emote{}

	for _, ae := range set.Emotes {
		e := utils.PointerOf(emoteMap[ae.ID])
		ae.Emote = e

		if ae.Emote == nil {
			continue
		}

		v := ae.Emote.GetLatestVersion(false)
		if v.ID.IsZero() || v.IsUnavailable() || v.IsProcessing() {
			continue
		}

		ae.Emote.Name = ae.Name
		result = append(result, model.NewEmote(*ae.Emote, r.Ctx.Config().CdnURL))
	}

	return ctx.JSON(rest.OK, result)
}
