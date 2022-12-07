package emote

import (
	"context"
	"strconv"
	"sync"

	"github.com/seventv/api/internal/api/gql/v2/gen/generated"
	"github.com/seventv/api/internal/api/gql/v2/gen/model"
	"github.com/seventv/api/internal/api/gql/v2/helpers"
	"github.com/seventv/api/internal/api/gql/v2/types"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Resolver struct {
	types.Resolver

	mx *sync.Mutex
}

func New(r types.Resolver) generated.EmoteResolver {
	return &Resolver{
		Resolver: r,
		mx:       &sync.Mutex{},
	}
}

const (
	EMOTE_CHANNEL_QUERY_SIZE_MOST = 50
	EMOTE_CHANNEL_QUERY_PAGE_CAP  = 500
)

func (r *Resolver) Channels(ctx context.Context, obj *model.Emote, pageArg *int, limitArg *int) ([]*model.UserPartial, error) {
	limit := EMOTE_CHANNEL_QUERY_SIZE_MOST
	if limitArg != nil {
		limit = *limitArg
	}

	if limit > EMOTE_CHANNEL_QUERY_SIZE_MOST {
		limit = EMOTE_CHANNEL_QUERY_SIZE_MOST
	} else if limit < 1 {
		return nil, errors.ErrInvalidRequest().SetDetail("limit cannot be less than 1")
	}

	page := 1

	if pageArg != nil {
		page = *pageArg
	}

	if page < 1 {
		page = 1
	}

	if page > EMOTE_CHANNEL_QUERY_PAGE_CAP {
		return nil, errors.ErrInvalidRequest().SetFields(errors.Fields{
			"PAGE":  strconv.Itoa(page),
			"LIMIT": strconv.Itoa(EMOTE_CHANNEL_QUERY_PAGE_CAP),
		}).SetDetail("No further pagination is allowed")
	}

	emoteID, err := primitive.ObjectIDFromHex(obj.ID)
	if err != nil {
		return nil, err
	}

	users, _, err := r.getChannels(ctx, emoteID, page, limit)
	if err != nil && !errors.Compare(err, errors.ErrNoItems()) {
		return nil, err
	}

	result := make([]*model.UserPartial, len(users))
	for i, u := range users {
		result[i] = helpers.UserStructureToPartialModel(helpers.UserStructureToModel(u, r.Ctx.Config().CdnURL))
	}

	return result, nil
}

func (r *Resolver) ChannelCount(ctx context.Context, obj *model.Emote) (int, error) {
	emoteID, err := primitive.ObjectIDFromHex(obj.ID)
	if err != nil {
		return 0, err
	}

	_, count, err := r.getChannels(ctx, emoteID, 1, 1)
	if err != nil && !errors.Compare(err, errors.ErrNoItems()) {
		return 0, err
	}

	return int(count), nil
}

func (r *Resolver) getChannels(ctx context.Context, emoteID primitive.ObjectID, page, limit int) ([]structures.User, int64, error) {
	r.mx.Lock()
	defer r.mx.Unlock()

	return r.Ctx.Inst().Query.EmoteChannels(ctx, emoteID, page, limit)
}
