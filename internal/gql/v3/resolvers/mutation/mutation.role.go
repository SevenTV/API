package mutation

import (
	"context"
	"strconv"

	"github.com/seventv/api/data/mutate"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) CreateRole(ctx context.Context, data model.CreateRoleInput) (*model.Role, error) {
	actor := auth.For(ctx)

	rb := structures.NewRoleBuilder(structures.Role{}).
		SetName(data.Name)

	if err := r.Ctx.Inst().Mutate.CreateRole(ctx, rb, mutate.RoleMutationOptions{
		Actor: &actor,
	}); err != nil {
		return nil, err
	}

	return r.Ctx.Inst().Modelizer.Role(rb.Role).GQL(), nil
}

func (r *Resolver) EditRole(ctx context.Context, roleID primitive.ObjectID, data model.EditRoleInput) (*model.Role, error) {
	actor := auth.For(ctx)

	roles, err := r.Ctx.Inst().Query.Roles(ctx, bson.M{"_id": roleID})
	if err != nil {
		return nil, err
	}

	if len(roles) == 0 {
		return nil, errors.ErrUnknownRole()
	}

	rb := structures.NewRoleBuilder(roles[0])

	if data.Name != nil {
		rb.SetName(*data.Name)
	}

	if data.Color != nil {
		c := *data.Color

		rb.SetColor(utils.Color(c))
	}

	if data.Allowed != nil {
		a, _ := strconv.Atoi(*data.Allowed)

		rb.SetAllowed(structures.RolePermission(a))
	}

	if data.Denied != nil {
		d, _ := strconv.Atoi(*data.Denied)

		rb.SetDenied(structures.RolePermission(d))
	}

	if err := r.Ctx.Inst().Mutate.EditRole(ctx, rb, mutate.RoleEditOptions{
		Actor:            &actor,
		OriginalPosition: roles[0].Position,
	}); err != nil {
		return nil, err
	}

	return r.Ctx.Inst().Modelizer.Role(rb.Role).GQL(), nil
}

func (r *Resolver) DeleteRole(ctx context.Context, roleID primitive.ObjectID) (string, error) {
	actor := auth.For(ctx)

	roles, err := r.Ctx.Inst().Query.Roles(ctx, bson.M{"_id": roleID})
	if err != nil {
		return "", err
	}

	if len(roles) == 0 {
		return "", errors.ErrUnknownRole()
	}

	if err := r.Ctx.Inst().Mutate.DeleteRole(ctx, structures.NewRoleBuilder(roles[0]), mutate.RoleMutationOptions{
		Actor: &actor,
	}); err != nil {
		return "", err
	}

	return "", nil
}
