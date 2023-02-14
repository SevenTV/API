package query

import (
	"context"
	"strings"

	"github.com/seventv/api/internal/api/gql/v2/gen/model"
	"github.com/seventv/api/internal/api/gql/v2/helpers"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) Emote(ctx context.Context, id string) (*model.Emote, error) {
	eid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.ErrBadObjectID()
	}

	emote, err := r.Ctx.Inst().Loaders.EmoteByID().Load(eid)
	if emote.ID.IsZero() || emote.ID == structures.DeletedEmote.ID {
		return nil, errors.ErrUnknownEmote()
	}

	return helpers.EmoteStructureToModel(emote, r.Ctx.Config().CdnURL), err
}

func (r *Resolver) SearchEmotes(
	ctx context.Context,
	queryArg string,
	limitArg *int,
	pageArg *int,
	pageSizeArg *int,
	submittedBy *string,
	globalStateArg *string,
	sortByArg *string,
	sortOrderArg *int,
	channel *string,
	filterArg *model.EmoteFilter,
) ([]*model.Emote, error) {
	notAvailable := errors.ErrInsufficientPrivilege().SetDetail("This endpoint is no longer available. Please use V3")

	if globalStateArg == nil {
		return nil, notAvailable
	}

	state := strings.ToUpper(*globalStateArg)

	if state != "ONLY" {
		return nil, errors.ErrInsufficientPrivilege().SetDetail("This endpoint is no longer available. Please use V3")
	}

	set, err := r.Ctx.Inst().Query.GlobalEmoteSet(ctx)
	if err != nil {
		return nil, errors.ErrInsufficientPrivilege().SetDetail("This endpoint is no longer available. Please use V3")
	}

	emoteIDs := utils.Map(set.Emotes, func(x structures.ActiveEmote) primitive.ObjectID {
		return x.ID
	})

	emotes, _ := r.Ctx.Inst().Loaders.EmoteByID().LoadAll(emoteIDs)

	result := make([]*model.Emote, len(emotes))

	for i, e := range emotes {
		result[i] = helpers.EmoteStructureToModel(e, r.Ctx.Config().CdnURL)
	}

	return result, nil
}
