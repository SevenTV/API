package emote

import (
	"context"

	"github.com/seventv/api/data/model/modelgql"
	"github.com/seventv/api/data/mutate"
	"github.com/seventv/api/internal/api/gql/v3/auth"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Merge implements generated.EmoteOpsResolver
func (r *ResolverOps) Merge(ctx context.Context, obj *model.EmoteOps, targetID primitive.ObjectID, reasonArg *string) (*model.Emote, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	if !actor.HasPermission(structures.RolePermissionEditAnyEmote) {
		return nil, errors.ErrInsufficientPrivilege()
	}

	emote, err := r.Ctx.Inst().Query.Emotes(ctx, bson.M{"versions.id": obj.ID}).First()
	if err != nil {
		return nil, err
	}

	ver, ind := emote.GetVersion(obj.ID)
	if ind < 0 {
		return nil, errors.ErrUnknownEmote()
	}

	eb := structures.NewEmoteBuilder(emote)

	targetEmote, err := r.Ctx.Inst().Query.Emotes(ctx, bson.M{"versions.id": targetID}).First()
	if err != nil {
		return nil, err
	}

	reason := ""
	if reasonArg != nil {
		reason = *reasonArg
	}

	if err := r.Ctx.Inst().Mutate.MergeEmote(ctx, eb, mutate.MergeEmoteOptions{
		Actor:          actor,
		NewEmote:       targetEmote,
		VersionID:      ver.ID,
		Reason:         reason,
		SkipValidation: false,
	}); err != nil {
		return nil, err
	}

	returnEmote, err := r.Ctx.Inst().Loaders.EmoteByID().Load(targetEmote.ID)
	if err != nil {
		return nil, err
	}

	return modelgql.EmoteModel(r.Ctx.Inst().Modelizer.Emote(returnEmote)), nil
}
