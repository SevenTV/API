package eventbridge

import (
	"context"
	"encoding/json"
	"time"

	"github.com/seventv/api/data/events"
	"github.com/seventv/api/data/model"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/svc/presences"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.uber.org/zap"
)

func handlePresence(gctx global.Context, ctx context.Context, cmd events.BridgedCommandPayload[events.PresenceCommandBody]) error {
	body := cmd.Body

	authentic := !cmd.ActorID.IsZero() && cmd.ActorID == body.UserID

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
		if user, err := gctx.Inst().Loaders.UserByConnectionID(pd.Platform).Load(pd.ID); err == nil && !user.ID.IsZero() {
			known = true
		}

		pm := gctx.Inst().Presences.ChannelPresence(ctx, cmd.Body.UserID)

		ttl := utils.Ternary(known, time.Hour*24, time.Minute*12) // set lower ttl for an unknown channel

		p, err := pm.Write(ctx, ttl, structures.UserPresenceDataChannel{
			Platform: pd.Platform,
			ID:       pd.ID,
			Filter:   pd.Filter,
		}, presences.WritePresenceOptions{
			IP:        cmd.ClientIP,
			Known:     known,
			Authentic: authentic,
		})

		if err != nil {
			return errors.From(err)
		}

		if err := gctx.Inst().Presences.ChannelPresenceFanout(ctx, presences.ChannelPresenceFanoutOptions{
			Presence: p,
			Passive:  body.Self,
			Whisper:  utils.Ternary(body.Self, cmd.SessionID, ""),
		}); err != nil {
			zap.S().Errorw("failed to fanout channel presence", "error", err)
		}
	}

	return nil
}
