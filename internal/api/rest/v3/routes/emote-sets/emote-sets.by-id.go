package emote_sets

import (
	"strings"

	"github.com/seventv/api/internal/api/rest/middleware"
	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type emoteSetByIDRoute struct {
	Ctx global.Context
}

func newEmoteSetByIDRoute(gctx global.Context) rest.Route {
	return &emoteSetByIDRoute{gctx}
}

func (r *emoteSetByIDRoute) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:      "/{emote-set.id}",
		Method:   rest.GET,
		Children: []rest.Route{},
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 60, []string{"public"}),
		},
	}
}

// @Summary Get Emote Set
// @Description Get an emote set by its ID
// @Tags emote-sets
// @Produce json
// @Param emote-set.id path string true "ID of the emote set"
// @Success 200 {object} model.EmoteSetModel
// @Router /emote-sets/{emote-set.id} [get]
func (r *emoteSetByIDRoute) Handler(ctx *rest.Ctx) rest.APIError {
	setID, err := ctx.UserValue("emote-set.id").ObjectID()
	if err != nil {
		if errors.Compare(err, errors.ErrBadObjectID()) {
			setName, _ := ctx.UserValue("emote-set.id").String()

			// Special named sets
			switch strings.ToUpper(setName) {
			case "GLOBAL":
				sys, err := r.Ctx.Inst().Mongo.System(ctx)
				if err != nil {
					if err == mongo.ErrNoDocuments {
						return errors.ErrUnknownEmoteSet()
					}

					return errors.ErrInternalServerError()
				}

				setID = sys.EmoteSetID
			}
		} else {
			return errors.From(err)
		}
	}

	set, err := r.Ctx.Inst().Loaders.EmoteSetByID().Load(setID)
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
		if emote.VersionRef == nil || emote.VersionRef.State.Lifecycle != structures.EmoteLifecycleLive {
			continue
		}

		emoteMap[emote.ID] = emote
	}

	setOwner, _ := r.Ctx.Inst().Loaders.UserByID().Load(set.OwnerID)
	if !setOwner.ID.IsZero() {
		set.Owner = &setOwner
	}

	emoteResult := make([]structures.ActiveEmote, len(set.Emotes))

	pos := 0

	for _, ae := range set.Emotes {
		e, ok := emoteMap[ae.ID]
		if !ok {
			continue
		}

		ae.Emote = &e
		emoteResult[pos] = ae

		pos++
	}

	set.Emotes = emoteResult[:pos]

	if set.ID.IsZero() {
		return errors.ErrUnknownEmoteSet()
	}

	return ctx.JSON(rest.OK, r.Ctx.Inst().Modelizer.EmoteSet(set))
}
