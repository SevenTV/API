package events

import (
	"context"
	"encoding/json"
	"hash/crc32"
	"time"

	"github.com/seventv/api/data/model"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.uber.org/zap"
)

type Instance interface {
	Dispatch(ctx context.Context, t EventType, cm ChangeMap, cond ...EventCondition)
	DispatchWithEffect(ctx context.Context, t EventType, cm ChangeMap, opt DispatchOptions, cond ...EventCondition) Message[DispatchPayload]
}

type eventsInst struct {
	ctx   context.Context
	redis redis.Instance

	dispatchQueue utils.Queue[Message[DispatchPayload]]
}

func NewPublisher(ctx context.Context, redis redis.Instance) Instance {
	ticker := time.NewTicker(50 * time.Millisecond)

	inst := &eventsInst{
		ctx:           ctx,
		redis:         redis,
		dispatchQueue: utils.NewQueue[Message[DispatchPayload]](10),
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if inst.dispatchQueue.IsEmpty() {
					continue
				}

				p := redis.RawClient().Pipeline()

				for _, m := range inst.dispatchQueue.Items() {
					j, err := json.Marshal(m)
					if err != nil {
						continue
					}

					k := CreateDispatchKey(m.Data.Type, m.Data.Conditions)

					p.Publish(ctx, k, j)
				}

				inst.dispatchQueue.Clear()

				if _, err := p.Exec(ctx); err != nil {
					zap.S().Warnw("failed to publish events",
						"error", err.Error(),
					)
				}
			}
		}
	}()

	return inst
}

// systemUser is a placeholder for the ChangeMap actor when no actor was provided
var systemUser = model.UserModel{
	ID:          structures.SystemUser.ID,
	UserType:    model.UserTypeModel(structures.SystemUser.UserType),
	Username:    structures.SystemUser.Username,
	DisplayName: structures.SystemUser.DisplayName,
}

func (inst *eventsInst) Dispatch(ctx context.Context, t EventType, cm ChangeMap, cond ...EventCondition) {
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

	inst.dispatchQueue.Add(msg)
}

func (inst *eventsInst) DispatchWithEffect(ctx context.Context, t EventType, cm ChangeMap, opt DispatchOptions, cond ...EventCondition) Message[DispatchPayload] {
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

	inst.dispatchQueue.Add(msg)

	return msg
}

type DispatchOptions struct {
	Whisper       string
	Effect        *SessionEffect
	DisableDedupe bool
}
