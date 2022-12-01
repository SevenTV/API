package query

import (
	"context"

	"github.com/seventv/api/internal/gql/v2/gen/model"
	"github.com/seventv/api/internal/gql/v2/helpers"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
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
	return nil, errors.ErrInsufficientPrivilege().SetDetail("This endpoint is no longer available. Please use V3")
}
