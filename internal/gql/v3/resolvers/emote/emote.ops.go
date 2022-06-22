package emote

import (
	"context"

	"github.com/seventv/api/internal/events"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/api/internal/gql/v3/types"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/mutations"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
)

type ResolverOps struct {
	types.Resolver
}

func NewOps(r types.Resolver) generated.EmoteOpsResolver {
	return &ResolverOps{r}
}

func (r *ResolverOps) Update(ctx context.Context, obj *model.EmoteOps, params model.EmoteUpdate) (*model.Emote, error) {
	actor := auth.For(ctx)
	if actor == nil {
		return nil, errors.ErrUnauthorized()
	}

	emotes, err := r.Ctx.Inst().Query.Emotes(ctx, bson.M{"versions.id": obj.ID}).Items()
	if err != nil {
		return nil, err
	}

	if len(emotes) == 0 {
		return nil, errors.ErrUnknownEmote()
	}

	emote := emotes[0]
	ver, _ := emote.GetVersion(obj.ID)
	eb := structures.NewEmoteBuilder(emote)

	// Cannot edit deleted version without privileges
	if !actor.HasPermission(structures.RolePermissionEditAnyEmote) && ver.IsUnavailable() {
		return nil, errors.ErrUnknownEmote()
	}

	if ver.IsProcessing() {
		return nil, errors.ErrInsufficientPrivilege().SetDetail("Cannot edit emote in a processing state")
	}

	// Edit name
	if params.Name != nil {
		eb.SetName(*params.Name)
	}
	// Edit owner
	if params.OwnerID != nil {
		eb.SetOwnerID(*params.OwnerID)
	}
	// Edit tags
	if params.Tags != nil {
		eb.SetTags(params.Tags, true)
	}
	// Edit flags
	if params.Flags != nil {
		f := structures.EmoteFlag(*params.Flags)
		eb.SetFlags(f)
	}

	// Edit listed (version)
	versionUpdated := false

	if params.Listed != nil {
		ver.State.Listed = *params.Listed
		versionUpdated = true
	}

	if params.VersionName != nil {
		ver.Name = *params.VersionName
		versionUpdated = true
	}

	if params.VersionDescription != nil {
		ver.Description = *params.VersionDescription
		versionUpdated = true
	}

	if params.Deleted != nil {
		ver.State.Lifecycle = utils.Ternary(*params.Deleted, structures.EmoteLifecycleDeleted, structures.EmoteLifecycleLive)
		versionUpdated = true
	}

	if versionUpdated {
		eb.UpdateVersion(obj.ID, ver)
	}

	if err := r.Ctx.Inst().Mutate.EditEmote(ctx, eb, mutations.EmoteEditOptions{
		Actor: actor,
	}); err != nil {
		return nil, err
	}

	go func() {
		events.Publish(r.Ctx, "emotes", obj.ID)
	}()

	emote, err = r.Ctx.Inst().Loaders.EmoteByID().Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return helpers.EmoteStructureToModel(emote, r.Ctx.Config().CdnURL), nil
}
