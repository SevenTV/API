package user

import (
	"context"
	"strings"

	"github.com/seventv/api/data/events"
	"github.com/seventv/api/internal/api/gql/v3/auth"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
)

func (r *ResolverOps) Cosmetics(ctx context.Context, obj *model.UserOps, update model.UserCosmeticUpdate) (*bool, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	if actor.ID != obj.ID && !actor.HasPermission(structures.RolePermissionManageUsers) {
		return nil, errors.ErrInsufficientPrivilege()
	}

	id := update.ID
	kind := update.Kind
	selected := update.Selected

	w := []mongo.WriteModel{}

	changeFields := []events.ChangeField{}

	// Get the cosmetic item
	cos := structures.Cosmetic[bson.Raw]{}
	if !id.IsZero() {
		if err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameCosmetics).FindOne(ctx, bson.M{
			"_id":  id,
			"kind": kind,
		}).Decode(&cos); err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, errors.ErrUnknownCosmetic()
			}

			r.Z().Errorw("failed to get cosmetic", "error", err)

			return nil, errors.ErrInternalServerError()
		}

		w = append(w, &mongo.UpdateOneModel{
			Filter: bson.M{
				"kind":     kind,
				"data.ref": cos.ID,
				"user_id":  obj.ID,
			},
			Update: bson.M{"$set": bson.M{"data.selected": selected}},
		})

		// Get user's current cosmetics
		ents, err := r.Ctx.Inst().Loaders.EntitlementsLoader().Load(obj.ID)
		if err != nil {
			r.Z().Errorw("failed to get user entitlements", "error", err)

			return nil, errors.ErrInternalServerError()
		}

		var (
			activeCos structures.Cosmetic[bson.Raw]
			oldData   any
			newData   any
		)

		switch kind {
		case model.CosmeticKindBadge:
			b, _, ok := ents.ActiveBadge()
			if ok {
				activeCos = b.ToRaw()
				oldData = r.Ctx.Inst().Modelizer.Badge(b)
			}

			if nb, err := structures.ConvertCosmetic[structures.CosmeticDataBadge](cos); err == nil {
				newData = r.Ctx.Inst().Modelizer.Badge(nb)
			}
		case model.CosmeticKindPaint:
			p, _, ok := ents.ActivePaint()
			if ok {
				activeCos = p.ToRaw()
				oldData = r.Ctx.Inst().Modelizer.Paint(p)
			}

			if np, err := structures.ConvertCosmetic[structures.CosmeticDataPaint](cos); err == nil {
				newData = r.Ctx.Inst().Modelizer.Paint(np)
			}
		}

		if !activeCos.ID.IsZero() && activeCos.ID != cos.ID {
			changeFields = append(changeFields, events.ChangeField{
				Key:    "style",
				Nested: true,
				Type:   events.ChangeFieldTypeObject,
				Value: []events.ChangeField{
					{
						Key:      strings.ToLower(kind.String()) + "_id",
						Type:     events.ChangeFieldTypeString,
						OldValue: activeCos.ID.Hex(),
						Value:    cos.ID.Hex(),
					},
					{
						Key:      strings.ToLower(kind.String()),
						Type:     events.ChangeFieldTypeObject,
						OldValue: oldData,
						Value:    newData,
					},
				},
			})
		}
	}

	w = append(w, &mongo.UpdateManyModel{
		Filter: bson.M{
			"kind":     kind,
			"data.ref": bson.M{"$not": bson.M{"$eq": cos.ID}},
			"user_id":  obj.ID,
		},
		Update: bson.M{
			"$set": bson.M{"data.selected": false},
		},
	})

	res, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEntitlements).BulkWrite(ctx, w)
	if err != nil {
		r.Z().Errorw("failed to update user cosmetic state", "error", err)

		return nil, errors.ErrInternalServerError()
	}

	if res.ModifiedCount > 0 {
		if err = r.Ctx.Inst().Events.Dispatch(ctx, events.EventTypeUpdateUser, events.ChangeMap{
			ID:      obj.ID,
			Kind:    structures.ObjectKindUser,
			Actor:   r.Ctx.Inst().Modelizer.User(actor).ToPartial(),
			Updated: changeFields,
		}, events.EventCondition{
			"object_id": obj.ID.Hex(),
		}); err != nil {
			r.Z().Errorw("failed to dispatch event", "error", err)
		}
	}

	return utils.PointerOf(true), nil
}
