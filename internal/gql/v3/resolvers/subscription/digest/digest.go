package digest

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"sync"

	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/common/events"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/sync_map"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type SubID [8]byte

var Mx *sync.Mutex = &sync.Mutex{}

var Chans *sync_map.Map[SubID, *ActiveSub] = &sync_map.Map[SubID, *ActiveSub]{}

func Digest(gctx global.Context) {
	chKey := redis.Key(events.OpcodeDispatch.PublishKey())

	ch := make(chan string, 10)

	go gctx.Inst().Redis.Subscribe(gctx, ch, chKey)
	go func() {
		defer close(ch)

		var (
			s   string
			err error
		)

		for {
			select {
			case <-gctx.Done():
				return
			case s = <-ch:
				o := events.Message[events.DispatchPayload]{}
				if err = json.Unmarshal(utils.S2B(s), &o); err != nil {
					zap.S().Warnw("got badly encoded message", "error", err.Error())
				}

				Chans.Range(func(key SubID, sub *ActiveSub) bool {
					if o.Data.Type != sub.Type {
						return true
					}

					cond := o.Data.Condition
					objectId, _ := primitive.ObjectIDFromHex(cond["object_id"])

					if objectId.IsZero() || objectId != sub.Target {
						return true
					}

					cm := o.Data.Body

					val := transformChangeMap(cm)

					select {
					case sub.Ch <- &val:
					default:
						zap.S().Warnw("channel blocked", "key", key)
					}

					return true
				})
			}
		}
	}()
}

func transformChangeMap(cm events.ChangeMap) model.ChangeMap {
	return model.ChangeMap{
		ID:      cm.ID,
		Kind:    model.ObjectKind(cm.Kind.String()),
		Actor:   &model.User{ID: cm.Actor.ID},
		Added:   transformChangeFields(cm.Added),
		Updated: transformChangeFields(cm.Updated),
		Removed: transformChangeFields(cm.Removed),
		Pushed:  transformChangeFields(cm.Pushed),
		Pulled:  transformChangeFields(cm.Pulled),
	}
}

func transformChangeFields(fields []events.ChangeField) []*model.ChangeField {
	result := make([]*model.ChangeField, len(fields))

	for i, cf := range fields {
		ind := 0
		if cf.Index != nil {
			ind = int(*cf.Index)
		}

		result[i] = &model.ChangeField{
			Key:      cf.Key,
			Index:    utils.PointerOf(ind),
			Nested:   cf.Nested,
			OldValue: encodeValue(cf.OldValue),
			Value:    encodeValue(cf.Value),
		}
	}

	return result
}

func encodeValue(v any) *string {
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)

	err := json.NewEncoder(encoder).Encode(v)
	if err != nil {
		return nil
	}

	encoder.Close()

	return utils.PointerOf(buf.String())
}

type ActiveSub struct {
	Ch     chan *model.ChangeMap
	Type   events.EventType
	Target primitive.ObjectID
}
