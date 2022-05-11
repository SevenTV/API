package events

import (
	"encoding/json"
	"fmt"

	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/Common/utils"
	"github.com/seventv/api/internal/global"
	v2helpers "github.com/seventv/api/internal/gql/v2/helpers"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Publish(ctx global.Context, objectType string, id primitive.ObjectID) {
	k := ctx.Inst().Redis.ComposeKey("events", fmt.Sprintf("sub:%s:%s", objectType, id.Hex()))
	ctx.Inst().Redis.RawClient().Publish(ctx, k.String(), "1")
}

func PublishLegacyEventAPI(
	ctx global.Context,
	action string,
	actor *structures.User,
	set structures.EmoteSet,
	emote structures.Emote,
	channelLogin string,
) {
	ae, _ := set.GetEmote(emote.ID)
	name := emote.Name
	if !ae.ID.IsZero() {
		name = ae.Name
	}

	evt := map[string]any{
		"channel":  set.Owner.Username,
		"emote_id": emote.ID,
		"name":     name,
		"action":   action,
		"actor":    actor.DisplayName,
	}
	if action != model.ListItemActionRemove.String() {
		e := v2helpers.EmoteStructureToModel(ctx, emote)
		var eOwner map[string]any
		if e.Owner != nil {
			eOwner = map[string]any{
				"id":           e.OwnerID,
				"twitch_id":    e.Owner.TwitchID,
				"display_name": e.Owner.DisplayName,
				"login":        e.Owner.Login,
			}
		}
		evt["emote"] = map[string]any{
			"name":       e.Name,
			"visibility": e.Visibility,
			"mime":       e.Mime,
			"tags":       e.Tags,
			"width":      e.Width,
			"height":     e.Height,
			"urls":       e.Urls,
			"owner":      eOwner,
		}
	}

	k := ctx.Inst().Redis.ComposeKey("events-v1", fmt.Sprintf("channel-emotes:%s", channelLogin))
	j, _ := json.Marshal(evt)
	ctx.Inst().Redis.RawClient().Publish(ctx, k.String(), utils.B2S(j))
}
