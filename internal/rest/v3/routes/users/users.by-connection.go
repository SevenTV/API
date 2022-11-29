package users

import (
	"strings"

	"github.com/seventv/api/data/model"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/middleware"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type userConnectionRoute struct {
	Ctx global.Context
}

func newUserConnection(gctx global.Context) rest.Route {
	return &userConnectionRoute{gctx}
}

func (r *userConnectionRoute) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:      "/{connection.platform}/{connection.id}",
		Method:   rest.GET,
		Children: []rest.Route{},
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 60, []string{"s-maxage=60"}),
		},
	}
}

// @Summary Get User Connection
// @Description Query for a user's connected account and its attached emote set
// @Param {connection.id} path string true "twitch, youtube or discord user ID"
// @Tags users
// @Produce json
// @Success 200 {object} model.UserConnectionModel
// @Router /users/{connection.platform}/{connection.id} [get]
func (r *userConnectionRoute) Handler(ctx *rest.Ctx) rest.APIError {
	// Retrieve the platform desired
	platformArg, ok := ctx.UserValue("connection.platform").String()
	if !ok {
		return errors.ErrInvalidRequest().SetDetail("connection.platform must be specified")
	}

	// Filter out unsupported platforms
	platform := structures.UserConnectionPlatform(strings.ToUpper(platformArg))
	switch platform {
	case structures.UserConnectionPlatformTwitch:
	case structures.UserConnectionPlatformYouTube:
	case structures.UserConnectionPlatformDiscord:
	default:
		return errors.ErrUnknownUserConnection().SetDetail("'%s' is not supported", platform)
	}

	// Retrieve specified connection id
	connID, ok := ctx.UserValue("connection.id").String()
	if !ok {
		return errors.ErrInvalidRequest().SetDetail("connection.id must be specified")
	}

	// Fetch user data
	user, err := r.Ctx.Inst().Loaders.UserByConnectionID(platform).Load(connID)
	if err != nil {
		return errors.From(err)
	}

	uc, i := user.Connections.Get(connID)
	if i == -1 {
		return errors.ErrUnknownUserConnection()
	}

	// Fetch Emote Set
	var emoteSetModel model.EmoteSetModel

	if !uc.EmoteSetID.IsZero() {
		set, err := r.Ctx.Inst().Loaders.EmoteSetByID().Load(uc.EmoteSetID)
		if err != nil && !errors.Compare(err, errors.ErrUnknownEmoteSet()) {
			return errors.From(err)
		}

		userIDs := make(utils.Set[primitive.ObjectID])
		userIDs.Add(set.OwnerID)

		emoteIDs := utils.Map(set.Emotes, func(a structures.ActiveEmote) primitive.ObjectID {
			return a.ID
		})

		emotes, _ := r.Ctx.Inst().Loaders.EmoteByID().LoadAll(emoteIDs)

		emoteMap := map[primitive.ObjectID]structures.Emote{}
		for _, emote := range emotes {
			emoteMap[emote.ID] = emote

			userIDs.Add(emote.OwnerID)
		}

		users, _ := r.Ctx.Inst().Loaders.UserByID().LoadAll(userIDs.Values())

		userMap := map[primitive.ObjectID]structures.User{}
		for _, user := range users {
			userMap[user.ID] = user
		}

		set.Owner = utils.PointerOf(userMap[set.OwnerID])

		for i, ae := range set.Emotes {
			e := utils.PointerOf(emoteMap[ae.ID])

			if u, ok := userMap[e.OwnerID]; ok {
				e.Owner = &u
			}

			set.Emotes[i].Emote = e
		}

		emoteSetModel = r.Ctx.Inst().Modelizer.EmoteSet(set)
	}

	// Construct the final response structure
	userModel := r.Ctx.Inst().Modelizer.User(user)
	userConnModel := r.Ctx.Inst().Modelizer.UserConnection(uc)
	userConnModel.User = &userModel

	if !emoteSetModel.ID.IsZero() {
		userConnModel.EmoteSet = &emoteSetModel
	}

	return ctx.JSON(rest.OK, userConnModel)
}
