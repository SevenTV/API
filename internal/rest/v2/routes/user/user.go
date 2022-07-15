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

type Route struct {
	Ctx global.Context
}

func New(gCtx global.Context) rest.Route {
	return &Route{gCtx}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/users/{user}",
		Method: rest.GET,
		Children: []rest.Route{
			newEmotes(r.Ctx),
		},
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 600, []string{"s-maxage=600"}),
		},
	}
}

// Get User
// @Summary Get User
// @Description Finds a user by its ID, Username or Twitch ID
// @Tags users
// @Param user path string false "User ID, Username or Twitch ID"
// @Produce json
// @Success 200 {object} model.User
// @Router /users/{user} [get]
func (r *Route) Handler(ctx *rest.Ctx) errors.APIError {
	key, _ := ctx.UserValue("user").String()

	var id primitive.ObjectID
	if primitive.IsValidObjectID(key) {
		id, _ = primitive.ObjectIDFromHex(key)
	}

	filter := utils.Ternary(id.IsZero(), bson.M{"$or": bson.A{
		bson.M{"connections.id": key},
		bson.M{"connections.data.login": strings.ToLower(key)},
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

	return ctx.JSON(rest.OK, model.NewUser(user))
}
