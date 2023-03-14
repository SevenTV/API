package eventbridge

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/seventv/api/data/events"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/utils"
	"go.uber.org/zap"
)

const SESSION_ID_KEY = utils.Key("session_id")

func handle(gctx global.Context, name string, body []byte) error {
	var err error

	req := getCommandBody[json.RawMessage](body)

	ctx, cancel := context.WithCancel(gctx)
	ctx = context.WithValue(ctx, SESSION_ID_KEY, req.SessionID)

	defer cancel()

	switch name {
	case "userstate", "cosmetics":
		err = handleUserState(gctx, ctx, getCommandBody[events.UserStateCommandBody](body))
	case "presence":
		err = handlePresence(gctx, ctx, getCommandBody[events.PresenceCommandBody](body))
	}

	return err
}

// The EventAPI Bridge allows passing commands from the eventapi via the websocket
func New(gctx global.Context) <-chan interface{} {
	// EventAPI Bridge
	go func() {
		ch := make(chan string, 1024)
		go gctx.Inst().Redis.Subscribe(gctx, ch, gctx.Inst().Redis.ComposeKey("eventapi", "bridge"))

		for {
			select {
			case <-gctx.Done():
				return
			case msg := <-ch:
				go func(m string) {
					sp := strings.SplitN(m, ":", 2)
					if len(sp) != 2 {
						zap.S().Errorw("invalid eventapi bridge message",
							"reason", "bad length",
							"msg", m,
						)

						return
					}

					cmd := sp[0]
					bodyStr := sp[1]

					var body json.RawMessage
					if err := json.Unmarshal(utils.S2B(bodyStr), &body); err != nil {
						zap.S().Errorw("invalid eventapi bridge message", "msg", m, "err", err)

						return
					}

					if err := handle(gctx, cmd, body); err != nil {
						zap.S().Errorw("eventapi bridge command failed", "cmd", cmd, "err", err)
					}
				}(msg)
			}
		}
	}()

	return nil
}

func getCommandBody[T events.BridgedCommandBody](body []byte) events.BridgedCommandPayload[T] {
	var result events.BridgedCommandPayload[T]

	if err := json.Unmarshal(body, &result); err != nil {
		zap.S().Errorw("invalid eventapi bridge message", "err", err)
	}

	return result
}
