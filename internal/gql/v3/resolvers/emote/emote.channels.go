package emote

import (
	"context"
	"strconv"

	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
)

const EMOTE_CHANNEL_QUERY_SIZE_MOST = 50
const EMOTE_CHANNEL_QUERY_PAGE_CAP = 500

func (r *Resolver) Channels(ctx context.Context, obj *model.Emote, pageArg *int, limitArg *int) (*model.UserSearchResult, error) {
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

	users, count, err := r.Ctx.Inst().Query.EmoteChannels(ctx, obj.ID, page, limit)
	if err != nil {
		return nil, err
	}

	models := make([]*model.User, len(users))
	for i, u := range users {
		if u.ID.IsZero() {
			u = structures.DeletedUser
		}
		models[i] = helpers.UserStructureToModel(r.Ctx, u)
	}

	results := model.UserSearchResult{
		Total: int(count),
		Items: models,
	}
	return &results, nil
}
