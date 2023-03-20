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
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
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
// @Success 200 {object} model.PresenceModel
// @Router /users/{user.id}/presences [post]
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

	clientIP := ctx.ClientIP()

	var presence structures.UserPresence[bson.Raw]

	switch body.Kind {
	case model.UserPresenceKindChannel:
		var pd structures.UserPresenceDataChannel

		if err := json.Unmarshal(body.Data, &pd); err != nil {
			return errors.ErrInvalidRequest().SetDetail("invalid or missing channel presence data: %s", err.Error())
		}

		if pd.ID == "" {
			return errors.ErrBadObjectID().SetDetail("missing ID in channel presence data")
		}

		var known bool
		if user, err := r.gctx.Inst().Loaders.UserByConnectionID(pd.Platform).Load(pd.ID); err == nil && !user.ID.IsZero() {
			known = true
		}

		pm := r.gctx.Inst().Presences.ChannelPresence(ctx, userID)

		ttl := utils.Ternary(known, time.Hour*24, time.Minute*12) // set lower ttl for an unknown channel

		var p structures.UserPresence[structures.UserPresenceDataChannel]

		p, err = pm.Write(ctx, ttl, structures.UserPresenceDataChannel{
			Platform: pd.Platform,
			ID:       pd.ID,
			Filter:   pd.Filter,
		}, presences.WritePresenceOptions{
			Authentic: authentic,
			Known:     known,
			IP:        clientIP,
			Passive:   body.Passive,
		})
		if err != nil {
			return errors.From(err)
		}

		presence = p.ToRaw()

		go func() {
			_ = r.gctx.Inst().Presences.ChannelPresenceFanout(ctx, presences.ChannelPresenceFanoutOptions{
				Presence: p,
				Whisper:  body.SessionID,
				Passive:  body.Passive,
			})
		}()
	}

	return ctx.JSON(rest.OK, r.gctx.Inst().Modelizer.Presence(presence))
}

type userPresenceWriteBody struct {
	Kind model.PresenceKind `json:"kind"`
	// if specified, fanout will only be sent to this event session
	SessionID string `json:"session_id"`
	// if true, the presence will not be written to the database or update entitlements
	Passive bool            `json:"passive"`
	Data    json.RawMessage `json:"data"`
}
