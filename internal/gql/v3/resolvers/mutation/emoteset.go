package mutation

import (
	"context"

	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/Common/structures/v3/mutations"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) EmoteSet(ctx context.Context, id primitive.ObjectID) (*model.EmoteSetOps, error) {
	return &model.EmoteSetOps{
		ID: id,
	}, nil
}

// CreateEmoteSet: create a new emote set
func (r *Resolver) CreateEmoteSet(ctx context.Context, input model.CreateEmoteSetInput) (*model.EmoteSet, error) {
	actor := auth.For(ctx)

	// Set up emote set builder
	isPrivileged := false
	if input.Privileged != nil && *input.Privileged {
		isPrivileged = true
	}
	b := structures.NewEmoteSetBuilder(structures.EmoteSet{}).
		SetName(input.Name).
		SetPrivileged(isPrivileged).
		SetOwnerID(actor.ID).
		SetEmoteSlots(250)

	// Execute mutation
	if err := r.Ctx.Inst().Mutate.CreateEmoteSet(ctx, b, mutations.EmoteSetMutationOptions{
		Actor: actor,
	}); err != nil {
		return nil, err
	}

	emoteSet, err := r.Ctx.Inst().Loaders.EmoteSetByID().Load(b.EmoteSet.ID)
	if err != nil {
		return nil, err
	}

	return helpers.EmoteSetStructureToModel(emoteSet, r.Ctx.Config().CdnURL), nil
}
