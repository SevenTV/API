package events

import (
	"encoding/json"
	"fmt"

	"github.com/seventv/api/internal/global"
	v2helpers "github.com/seventv/api/internal/gql/v2/helpers"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Publish(ctx global.Context, objectType string, id primitive.ObjectID) {
	k := ctx.Inst().Redis.ComposeKey("events", fmt.Sprintf("sub:%s:%s", objectType, id.Hex()))
	ctx.Inst().Redis.RawClient().Publish(ctx, k.String(), "1")
}

type Event struct {
	Channel string               `json:"channel"`
	EmoteID primitive.ObjectID   `json:"emote_id"`
	Name    string               `json:"name"`
	Action  model.ListItemAction `json:"action"`
	Actor   string               `json:"actor"`
	Emote   EventEmote           `json:"emote"`
}

type EventEmote struct {
	Name       string          `json:"name"`
	Visibility int             `json:"visibility"`
	Mime       string          `json:"mime"`
	Tags       []string        `json:"tags"`
	Width      []int           `json:"width"`
	Height     []int           `json:"height"`
	Urls       [][]string      `json:"urls"`
	Owner      EventEmoteOwner `json:"owner"`
}

type EventEmoteOwner struct {
	ID          string `json:"id"`
	TwitchID    string `json:"twitch_id"`
	DisplayName string `json:"display_name"`
	Login       string `json:"login"`
}

func PublishLegacyEventAPI(
	ctx global.Context,
	action model.ListItemAction,
	channelLogin string,
	actor structures.User,
	set structures.EmoteSet,
	emote structures.Emote,
) error {
	ae, _ := set.GetEmote(emote.ID)
	name := emote.Name
	if !ae.ID.IsZero() {
		name = ae.Name
	}

	if set.Owner == nil {
		return nil
	}
	evt := Event{
		Channel: set.Owner.Username,
		EmoteID: emote.ID,
		Name:    name,
		Action:  action,
		Actor:   actor.DisplayName,
	}
	if action != model.ListItemActionRemove {
		e := v2helpers.EmoteStructureToModel(emote, ctx.Config().CdnURL)
		evt.Emote = EventEmote{
			Name:       e.Name,
			Visibility: e.Visibility,
			Mime:       e.Mime,
			Tags:       e.Tags,
			Width:      e.Width,
			Height:     e.Height,
			Urls:       e.Urls,
		}
		if e.Owner != nil {
			evt.Emote.Owner = EventEmoteOwner{
				ID:          e.OwnerID,
				TwitchID:    e.Owner.TwitchID,
				DisplayName: e.Owner.DisplayName,
				Login:       e.Owner.Login,
			}
		}
	}

	k := ctx.Inst().Redis.ComposeKey("events-v1", fmt.Sprintf("channel-emotes:%s", channelLogin))
	j, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	return ctx.Inst().Redis.RawClient().Publish(ctx, k.String(), utils.B2S(j)).Err()
}
