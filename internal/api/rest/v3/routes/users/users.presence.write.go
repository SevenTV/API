package users

import (
	"encoding/json"
	"time"

	"github.com/seventv/api/data/model"
	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/svc/presences"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
)

type userPresenceWriteRoute struct {
	gctx global.Context
}

func newUserPresenceWriteRoute(gctx global.Context) *userPresenceWriteRoute {
	return &userPresenceWriteRoute{
		gctx: gctx,
	}
}

func (r *userPresenceWriteRoute) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/{user.id}/presences",
		Method: rest.POST,
	}
}

// @Summary Update User Presence
// @Description Update user presence
// @Param userID path string true "ID of the user"
// @Tags users
// @Produce json
// @Success 200 {object} model.MutationResponse
// @Router /users/{user.id}/presence [post]
func (r *userPresenceWriteRoute) Handler(ctx *rest.Ctx) rest.APIError {
	var body userPresenceWriteBody

	userID, err := ctx.UserValue("user.id").ObjectID()
	if err != nil {
		return errors.From(err)
	}

	actor, ok := ctx.GetActor()

	authentic := ok && actor.ID == userID

	if err := json.Unmarshal(ctx.Request.Body(), &body); err != nil {
		return errors.ErrInvalidRequest()
	}

	clientIP, _ := ctx.UserValue(rest.ClientIP).String()

	switch body.Kind {
	case model.UserPresenceKindChannel:
		var pd structures.UserPresenceDataChannel

		if err := json.Unmarshal(body.Data, &pd); err != nil {
			return errors.ErrInvalidRequest().SetDetail("invalid or missing channel presence data: %s", err.Error())
		}

		if pd.ConnectionID == "" {
			return errors.ErrBadObjectID().SetDetail("missing connection ID")
		}

		if pd.HostID.IsZero() {
			return errors.ErrBadObjectID().SetDetail("invalid or missing host ID")
		}

		// Validate host user & connection (channel)
		user, err := r.gctx.Inst().Loaders.UserByID().Load(pd.HostID)
		if err != nil {
			return errors.From(err).SetDetail("Host")
		}

		uc, ind := user.Connections.Get(pd.ConnectionID)
		if ind == -1 {
			return errors.ErrUnknownUser().SetDetail("Host Connection")
		}

		pm := r.gctx.Inst().Presences.ChannelPresence(ctx, userID)

		if err := pm.Write(ctx, time.Minute*5, structures.UserPresenceDataChannel{
			HostID:       user.ID,
			ConnectionID: uc.ID,
		}, presences.WritePresenceOptions{
			Authentic: authentic,
			IP:        clientIP,
		}); err != nil {
			return errors.From(err)
		}
	}

	return ctx.JSON(rest.OK, model.MutationResponse{
		OK: true,
	})
}

type userPresenceWriteBody struct {
	Kind model.UserPresenceKind `json:"kind"`
	Data json.RawMessage        `json:"data"`
}
