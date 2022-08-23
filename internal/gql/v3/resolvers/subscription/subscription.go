package subscription

import (
	"context"
	"fmt"

	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/resolvers/subscription/digest"
	"github.com/seventv/api/internal/gql/v3/types"
	"github.com/seventv/common/events"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.SubscriptionResolver {
	return &Resolver{
		Resolver: r,
	}
}

func (r *Resolver) Z() *zap.SugaredLogger {
	return zap.S().Named("subscription")
}

func (r *Resolver) subscribe(ctx context.Context, objectType string, id primitive.ObjectID) <-chan string {
	chKey := r.Ctx.Inst().Redis.ComposeKey("events", fmt.Sprintf("sub:%s:%s", objectType, id.Hex()))
	subCh := make(chan string, 1)

	go r.Ctx.Inst().Redis.Subscribe(ctx, subCh, chKey)

	go func() {
		<-ctx.Done()
		close(subCh)
	}()

	return subCh
}

func (r *Resolver) subscribeNext(ctx context.Context, eventName events.EventType, id primitive.ObjectID) <-chan *model.ChangeMap {
	ch := make(chan *model.ChangeMap, 1)

	subIDVal, _ := utils.GenerateRandomBytes(8)
	subID := digest.SubID{}
	copy(subID[:], subIDVal)

	// Get existing state
	sub := &digest.ActiveSub{
		Ch:     ch,
		Type:   eventName,
		Target: id,
	}

	digest.Chans.Store(subID, sub)

	go func() {
		<-ctx.Done()

		close(ch)
		digest.Chans.Delete(subID)
	}()

	return ch
}
