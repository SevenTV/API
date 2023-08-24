package events

import (
	"encoding/json"
	"hash/crc32"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.uber.org/zap"

	"github.com/seventv/api/data/model"
)

type Instance interface {
	Dispatch(t EventType, cm ChangeMap, cond ...EventCondition)
	DispatchWithEffect(t EventType, cm ChangeMap, opt DispatchOptions, cond ...EventCondition) Message[DispatchPayload]
}

type EventsInst struct {
	nc      *nats.Conn
	subject string
}

func NewPublisher(nc *nats.Conn, subject string) Instance {
	return &EventsInst{
		nc:      nc,
		subject: subject,
	}
}

// systemUser is a placeholder for the ChangeMap actor when no actor was provided
var systemUser = model.UserModel{
	ID:          structures.SystemUser.ID,
	UserType:    model.UserTypeModel(structures.SystemUser.UserType),
	Username:    structures.SystemUser.Username,
	DisplayName: structures.SystemUser.DisplayName,
}

func (inst *EventsInst) Dispatch(t EventType, cm ChangeMap, cond ...EventCondition) {
	if cm.Actor.ID.IsZero() {
		cm.Actor = systemUser.ToPartial()
	}

	// Dedupe hash
	var dedupeHash *uint32

	if cm.Object != nil {
		h := crc32.New(crc32.MakeTable(2596996162))

		h.Write(cm.ID[:])
		h.Write(utils.S2B(cm.Kind.String()))
		h.Write(cm.Object)

		dedupeHash = utils.PointerOf(h.Sum32())
	}

	msg := NewMessage(OpcodeDispatch, DispatchPayload{
		Type:       t,
		Body:       cm,
		Hash:       dedupeHash,
		Conditions: cond,
	})

	data, err := json.Marshal(msg)
	if err != nil {
		zap.S().Warnw("failed to marshal event",
			"error", err.Error(),
		)

		return
	}

	for _, c := range cond {
		for _, b := range []bool{false, true} {
			err = inst.nc.Publish(inst.subject+"."+CreateDispatchKey(t, c, b), data)
			if err != nil {
				zap.S().Errorw("nats publish", "error", err)
			}
		}
	}
}

func (inst *EventsInst) DispatchWithEffect(t EventType, cm ChangeMap, opt DispatchOptions, cond ...EventCondition) Message[DispatchPayload] {
	if cm.Actor.ID.IsZero() {
		cm.Actor = systemUser.ToPartial()
	}

	// Dedupe hash
	var dedupeHash *uint32

	if !opt.DisableDedupe && cm.Object != nil {
		h := crc32.New(crc32.MakeTable(2596996162))

		h.Write(cm.ID[:])
		h.Write(utils.S2B(cm.Kind.String()))
		h.Write(cm.Object)

		dedupeHash = utils.PointerOf(h.Sum32())
	}

	msg := NewMessage(OpcodeDispatch, DispatchPayload{
		Type:       t,
		Body:       cm,
		Hash:       dedupeHash,
		Conditions: cond,
		Effect:     opt.Effect,
		Whisper:    opt.Whisper,
	})

	data, err := json.Marshal(msg)
	if err != nil {
		zap.S().Warnw("failed to marshal event",
			"error", err.Error(),
		)

		return msg
	}

	payloads := make(map[string][]byte)

	if opt.Whisper == "" {
		for _, c := range cond {
			for _, b := range []bool{false, true} {
				payloads[CreateDispatchKey(t, c, b)] = data
			}
		}
	} else {
		payloads[CreateDispatchKey(EventTypeWhisper, EventCondition{"session_id": opt.Whisper}, false)] = data
	}

	go func() {
		if opt.Delay > 0 {
			<-time.After(opt.Delay)
		}

		for key, data := range payloads {
			err = inst.nc.Publish(inst.subject+"."+key, data)
			if err != nil {
				zap.S().Errorw("nats publish", "error", err)
			}
		}
	}()

	return msg
}

type DispatchOptions struct {
	Delay         time.Duration
	Whisper       string
	Effect        *SessionEffect
	DisableDedupe bool
}
