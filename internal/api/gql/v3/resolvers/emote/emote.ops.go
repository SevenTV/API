package emote

import (
	"context"

	"github.com/seventv/api/data/model/modelgql"
	"github.com/seventv/api/data/mutate"
	"github.com/seventv/api/internal/api/gql/v3/auth"
	"github.com/seventv/api/internal/api/gql/v3/gen/generated"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/api/internal/api/gql/v3/types"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
)

type ResolverOps struct {
	types.Resolver
}

func NewOps(r types.Resolver) generated.EmoteOpsResolver {
	return &ResolverOps{r}
}

func (r *ResolverOps) Update(ctx context.Context, obj *model.EmoteOps, params model.EmoteUpdate, reason *string) (*model.Emote, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
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

	// Edit listed (version)
	versionUpdated := false

	// Reason
	rsn := ""
	if reason != nil {
		rsn = *reason
	}

	// Delete emote
	// no other params can be used if `deleted` is true
	if params.Deleted != nil {
		del := *params.Deleted

		err = r.Ctx.Inst().Mutate.DeleteEmote(ctx, eb, mutate.DeleteEmoteOptions{
			Actor:     actor,
			VersionID: obj.ID,
			Undo:      !del,
			Reason:    rsn,
		})

		if err != nil {
			return nil, err
		}
	} else {
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
			if !actor.HasPermission(structures.RolePermissionManageContent) {
				for _, tag := range params.Tags {
					if utils.Contains(r.Ctx.Config().Limits.Emotes.ReservedTags, tag) {
						return nil, errors.ErrInsufficientPrivilege().SetDetail("You cannot use reserved tag #%s", tag)
					}
				}
			}

			eb.SetTags(params.Tags, true)
		}
		// Edit flags
		if params.Flags != nil {
			f := structures.BitField[structures.EmoteFlag](structures.EmoteFlag(*params.Flags))
			eb.SetFlags(f)
		}

		if params.Listed != nil {
			ver.State.Listed = *params.Listed
			versionUpdated = true
		}

		if params.PersonalUse != nil {
			ver.State.AllowPersonal = params.PersonalUse
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

		if versionUpdated {
			eb.UpdateVersion(obj.ID, ver)
		}

		if err := r.Ctx.Inst().Mutate.EditEmote(ctx, eb, mutate.EmoteEditOptions{
			Actor: actor,
		}); err != nil {
			return nil, err
		}
	}

	emote, err = r.Ctx.Inst().Loaders.EmoteByID().Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return modelgql.EmoteModel(r.Ctx.Inst().Modelizer.Emote(emote)), nil
}
