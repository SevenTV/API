package events

import (
	"context"
	"encoding/json"
	"hash/crc32"
	"time"

	"github.com/seventv/api/data/model"
	"github.com/seventv/common/dataloader"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.uber.org/zap"
)

type Instance interface {
	Dispatch(ctx context.Context, t EventType, cm ChangeMap, cond ...EventCondition)
	DispatchWithEffect(ctx context.Context, t EventType, cm ChangeMap, opt DispatchOptions, cond ...EventCondition) Message[DispatchPayload]
}

type DataloaderPayload struct {
	Key  string
	Data string
}

type eventsInst struct {
	ctx   context.Context
	redis redis.Instance

	dl *dataloader.DataLoader[DataloaderPayload, struct{}]
}

func NewPublisher(ctx context.Context, redis redis.Instance) Instance {
	inst := &eventsInst{
		ctx:   ctx,
		redis: redis,
		dl: dataloader.New(dataloader.Config[DataloaderPayload, struct{}]{
			Fetch: func(keys []DataloaderPayload) ([]struct{}, []error) {
				p := redis.RawClient().Pipeline()

				for _, k := range keys {
					p.Publish(ctx, k.Key, k.Data)
				}

				if _, err := p.Exec(ctx); err != nil {
					zap.S().Warnw("failed to publish events",
						"error", err.Error(),
					)
				}

				return make([]struct{}, len(keys)), nil
			},
			Wait:     time.Duration(100) * time.Millisecond,
			MaxBatch: 128,
		}),
	}

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

	j, err := json.Marshal(msg)
	if err != nil {
		zap.S().Warnw("failed to marshal event",
			"error", err.Error(),
		)

		return
	}

	payloads := make([]DataloaderPayload, len(cond))
	s := utils.B2S(j)

	for i, c := range cond {
		payloads[i] = DataloaderPayload{
			Key:  CreateDispatchKey(t, c),
			Data: s,
		}
	}

	go func() {
		for _, p := range payloads {
			_, _ = inst.dl.Load(p)
		}
	}()
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

	j, err := json.Marshal(msg)
	if err != nil {
		zap.S().Warnw("failed to marshal event",
			"error", err.Error(),
		)

		return msg
	}

	payloads := make([]DataloaderPayload, len(cond))
	s := utils.B2S(j)

	for i, c := range cond {
		payloads[i] = DataloaderPayload{
			Key:  CreateDispatchKey(t, c),
			Data: s,
		}
	}

	go func() {
		if opt.Delay > 0 {
			<-time.After(opt.Delay)
		}

		for _, p := range payloads {
			_, _ = inst.dl.Load(p)
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
