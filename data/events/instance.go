package events

import (
	"context"
	"encoding/json"
	"hash/crc32"
	"strings"

	"github.com/seventv/api/data/model"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
)

type Instance interface {
	Publish(ctx context.Context, msg Message[json.RawMessage]) error
	Dispatch(ctx context.Context, t EventType, cm ChangeMap, cond ...EventCondition) error
	DispatchWithEffect(ctx context.Context, t EventType, cm ChangeMap, opt DispatchOptions, cond ...EventCondition) (Message[DispatchPayload], error)
}

type eventsInst struct {
	ctx   context.Context
	redis redis.Instance
}

func NewPublisher(ctx context.Context, redis redis.Instance) Instance {
	return &eventsInst{
		ctx:   ctx,
		redis: redis,
	}
}

func (inst *eventsInst) Publish(ctx context.Context, msg Message[json.RawMessage]) error {
	j, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	k := inst.redis.ComposeKey("events", "op", strings.ToLower(msg.Op.String()))
	if _, err = inst.redis.RawClient().Publish(ctx, k.String(), j).Result(); err != nil {
		return err
	}

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
	})

	return msg, inst.Publish(ctx, msg.ToRaw())
}

type DispatchOptions struct {
	Effect        *SessionEffect
	DisableDedupe bool
}
