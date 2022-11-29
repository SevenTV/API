package mutation

import (
	"context"

	"github.com/seventv/api/data/mutate"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) EmoteSet(ctx context.Context, id primitive.ObjectID) (*model.EmoteSetOps, error) {
	return &model.EmoteSetOps{
		ID: id,
	}, nil
}

// CreateEmoteSet: create a new emote set
func (r *Resolver) CreateEmoteSet(ctx context.Context, userID primitive.ObjectID, input model.CreateEmoteSetInput) (*model.EmoteSet, error) {
	actor := auth.For(ctx)

	// Set up emote set builder
	isPrivileged := false
	if input.Privileged != nil && *input.Privileged {
		isPrivileged = true
	}

	b := structures.NewEmoteSetBuilder(structures.EmoteSet{Emotes: []structures.ActiveEmote{}}).
		SetName(input.Name).
		SetPrivileged(isPrivileged).
		SetOwnerID(userID).
		SetCapacity(300)

	// Execute mutation
	if err := r.Ctx.Inst().Mutate.CreateEmoteSet(ctx, b, mutate.EmoteSetMutationOptions{
		Actor: actor,
	}); err != nil {
		return nil, err
	}

	emoteSet, err := r.Ctx.Inst().Query.EmoteSets(ctx, bson.M{"_id": b.EmoteSet.ID}).First()
	if err != nil {
		return nil, err
	}

	return r.Ctx.Inst().Modelizer.EmoteSet(emoteSet).GQL(), nil
}
