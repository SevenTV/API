package eventbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/seventv/api/data/events"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/utils"
	"go.uber.org/zap"
)

const SESSION_ID_KEY = utils.Key("session_id")

func handle(gctx global.Context, body []byte) error {
	var err error

	req := getCommandBody[json.RawMessage](body)

	ctx, cancel := context.WithCancel(gctx)
	ctx = context.WithValue(ctx, SESSION_ID_KEY, req.SessionID)

	defer cancel()

	switch req.Command {
	case "userstate", "cosmetics":
		data := getCommandBody[events.UserStateCommandBody](body).Body

		err = handleUserState(gctx, ctx, data)
	}

	return err
}

// The EventAPI Bridge allows passing commands from the eventapi via the websocket
func New(gctx global.Context) <-chan struct{} {
	done := make(chan struct{})

	createUserStateLoader(gctx)

	go func() {
		err := http.ListenAndServe(gctx.Config().EventBridge.Bind, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var err error

			// read body into byte slice
			if r.Body == nil {
				zap.S().Errorw("invalid eventapi bridge message", "err", "empty body")
			}

			defer r.Body.Close()

			var buf bytes.Buffer
			if _, err = buf.ReadFrom(r.Body); err != nil {
				zap.S().Errorw("invalid eventapi bridge message", "err", err)

				return
			}

			if err := handle(gctx, buf.Bytes()); err != nil {
				zap.S().Errorw("eventapi bridge command failed", "error", err)
			}
		}))

		if err != nil {
			zap.S().Errorw("eventapi bridge failed", "error", err)

			close(done)
		}
	}()

	go func() {
		<-gctx.Done()
		close(done)
	}()

	return done
}

func getCommandBody[T events.BridgedCommandBody](body []byte) events.BridgedCommandPayload[T] {
	var result events.BridgedCommandPayload[T]

	if err := json.Unmarshal(body, &result); err != nil {
		zap.S().Errorw("invalid eventapi bridge message", "err", err)
	}

	return result
}
