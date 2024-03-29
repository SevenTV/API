package users

import (
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/data/model"
	"github.com/seventv/api/internal/api/rest/middleware"
	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type userRoute struct {
	Ctx global.Context
}

func newUser(gctx global.Context) rest.Route {
	return &userRoute{gctx}
}

func (r *userRoute) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:      "/{user.id}",
		Method:   rest.GET,
		Children: []rest.Route{},
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 300, []string{"public"}),
		},
	}
}

// @Summary Get User
// @Description Get user by ID
// @Param userID path string true "ID of the user"
// @Tags users
// @Produce json
// @Success 200 {object} model.UserModel
// @Router /users/{user.id} [get]
func (r *userRoute) Handler(ctx *rest.Ctx) rest.APIError {
	userID, err := ctx.UserValue("user.id").ObjectID()
	if err != nil {
		return errors.From(err)
	}

	user, err := r.Ctx.Inst().Loaders.UserByID().Load(userID)
	if err != nil {
		return errors.From(err)
	}

	var (
		result         = r.Ctx.Inst().Modelizer.User(user)
		entitledSetIDs []primitive.ObjectID
		sets           []structures.EmoteSet
	)

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()

		entitledSetIDs = utils.Map(userWithEntitledEmoteSets(r.Ctx, user), func(x model.EmoteSetPartialModel) primitive.ObjectID {
			return x.ID
		})
	}()

	go func() {
		defer wg.Done()

		sets, err = r.Ctx.Inst().Loaders.EmoteSetByUserID().Load(user.ID)
		if err != nil {
			zap.S().Errorw("failed to load emote sets of user", "error", err)
		}
	}()

	wg.Wait()

	if len(sets) > 0 {
		result.EmoteSets = []model.EmoteSetPartialModel{}

		for _, set := range sets {
			if set.Flags.Has(structures.EmoteSetFlagPersonal) && !utils.Contains(entitledSetIDs, set.ID) {
				continue
			}

			set.OwnerID = primitive.NilObjectID
			set.Emotes = nil

			result.EmoteSets = append(result.EmoteSets, r.Ctx.Inst().Modelizer.EmoteSet(set).ToPartial())
		}
	}

	return ctx.JSON(rest.OK, result)
}

func userWithEntitledEmoteSets(gctx global.Context, user structures.User) []model.EmoteSetPartialModel {
	ents, err := gctx.Inst().Loaders.EntitlementsLoader().Load(user.ID)
	if err != nil {
		return nil
	}

	if len(ents.EmoteSets) == 0 {
		return nil
	}

	setIDs := utils.Map(ents.EmoteSets, func(x structures.Entitlement[structures.EntitlementDataEmoteSet]) primitive.ObjectID {
		return x.Data.RefID
	})

	result := make([]model.EmoteSetPartialModel, len(ents.EmoteSets))

	sets, errs := gctx.Inst().Loaders.EmoteSetByID().LoadAll(setIDs)
	if err = multierror.Append(nil, errs...).ErrorOrNil(); err != nil {
		return nil
	}

	for i, set := range sets {
		v := gctx.Inst().Modelizer.EmoteSet(set).ToPartial()
		v.Owner = nil

		result[i] = v
	}

	return result
}
