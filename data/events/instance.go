package events

import (
	"context"
	"encoding/json"
	"hash/crc32"
	"strings"
	"time"

	"github.com/seventv/api/data/model"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.uber.org/zap"
)

type Instance interface {
	Publish(ctx context.Context, msg Message[json.RawMessage]) error
	Dispatch(ctx context.Context, t EventType, cm ChangeMap, cond ...EventCondition) error
	DispatchWithEffect(ctx context.Context, t EventType, cm ChangeMap, opt DispatchOptions, cond ...EventCondition) (Message[DispatchPayload], error)
}

type eventsInst struct {
	ctx   context.Context
	redis redis.Instance

	publishQueue utils.Queue[Message[json.RawMessage]]
}

func NewPublisher(ctx context.Context, redis redis.Instance) Instance {
	ticker := time.NewTicker(50 * time.Millisecond)

	inst := &eventsInst{
		ctx:          ctx,
		redis:        redis,
		publishQueue: utils.NewQueue[Message[json.RawMessage]](10),
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if inst.publishQueue.IsEmpty() {
					continue
				}

				p := redis.RawClient().Pipeline()

				for _, m := range inst.publishQueue.Items() {
					j, err := json.Marshal(m)
					if err != nil {
						continue
					}

					k := redis.ComposeKey("events", "op", strings.ToLower(m.Op.String()))
					p.Publish(ctx, k.String(), j)
				}

				inst.publishQueue.Clear()

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

func (inst *eventsInst) Publish(ctx context.Context, msg Message[json.RawMessage]) error {
	inst.publishQueue.Add(msg)

	return nil
}

// systemUser is a placeholder for the ChangeMap actor when no actor was provided
var systemUser = model.UserModel{
	ID:          structures.SystemUser.ID,
	UserType:    model.UserTypeModel(structures.SystemUser.UserType),
	Username:    structures.SystemUser.Username,
	DisplayName: structures.SystemUser.DisplayName,
}

func (inst *eventsInst) Dispatch(ctx context.Context, t EventType, cm ChangeMap, cond ...EventCondition) error {
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

	return inst.Publish(ctx, msg.ToRaw())
}

func (inst *eventsInst) DispatchWithEffect(ctx context.Context, t EventType, cm ChangeMap, opt DispatchOptions, cond ...EventCondition) (Message[DispatchPayload], error) {
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

	return msg, inst.Publish(ctx, msg.ToRaw())
}

type DispatchOptions struct {
	Whisper       string
	Effect        *SessionEffect
	DisableDedupe bool
}
