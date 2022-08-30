package mutate

import (
	"context"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// Create: create the new role
func (m *Mutate) CreateRole(ctx context.Context, rb *structures.RoleBuilder, opt RoleMutationOptions) error {
	if rb == nil {
		return errors.ErrInternalIncompleteMutation()
	}

	if rb.Role.Name == "" {
		return errors.ErrValidationRejected().SetDetail("Missing role name")
	}

	// Check actor's permissions
	if opt.Actor != nil && !opt.Actor.HasPermission(structures.RolePermissionManageRoles) {
		return errors.ErrInsufficientPrivilege()
	}

	// Create the role
	rb.Role.ID = primitive.NewObjectID()
	result, err := m.mongo.Collection(mongo.CollectionNameRoles).InsertOne(ctx, rb.Role)

	if err != nil {
		zap.S().Errorw("mongo, error while writing new role to database", "error", err)
		return errors.ErrInternalServerError()
	}

	// Get the newly created role
	if err = m.mongo.Collection(mongo.CollectionNameRoles).FindOne(ctx, bson.M{"_id": result.InsertedID}).Decode(&rb.Role); err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.ErrUnknownRole()
		}

		return errors.ErrInternalServerError()
	}

	return nil
}

// Edit: edit the role. Modify the RoleBuilder beforehand!
func (m *Mutate) EditRole(ctx context.Context, rb *structures.RoleBuilder, opt RoleEditOptions) error {
	if rb == nil {
		return errors.ErrInternalIncompleteMutation()
	}

	// Check actor's permissions
	actor := opt.Actor
	if actor != nil {
		if !actor.HasPermission(structures.RolePermissionManageRoles) {
			return errors.ErrInsufficientPrivilege()
		}

		if len(opt.Actor.Roles) > 0 {
			// ensure that the actor's role is higher than the role being deleted
			highestRole := actor.GetHighestRole()
			if opt.OriginalPosition >= highestRole.Position {
				return errors.ErrInsufficientPrivilege()
			}
		}
	}

	// Update the role
	if err := m.mongo.Collection(mongo.CollectionNameRoles).FindOneAndUpdate(
		ctx,
		bson.M{"_id": rb.Role.ID},
		rb.Update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&rb.Role); err != nil {
		zap.S().Errorw("mongo, error while updating role in database")

		return errors.ErrInternalServerError()
	}

	return nil
}

// Delete: delete the role
func (m *Mutate) DeleteRole(ctx context.Context, rb *structures.RoleBuilder, opt RoleMutationOptions) error {
	if rb == nil {
		return errors.ErrInternalIncompleteMutation()
	}

	// Check actor's permissions
	actor := opt.Actor
	if actor != nil {
		if !actor.HasPermission(structures.RolePermissionManageRoles) {
			return errors.ErrInsufficientPrivilege()
		}

		if len(opt.Actor.Roles) > 0 {
			// ensure that the actor's role is higher than the role being deleted
			actor.SortRoles()
			highestRole := actor.Roles[0]

			if rb.Role.Position >= highestRole.Position {
				return errors.ErrInsufficientPrivilege()
			}
		}
	}

	// Delete the role
	if _, err := m.mongo.Collection(mongo.CollectionNameRoles).DeleteOne(ctx, bson.M{"_id": rb.Role.ID}); err != nil {
		return err
	}

	// Remove the role from any user who had it
	_, err := m.mongo.Collection(mongo.CollectionNameUsers).UpdateMany(ctx, bson.M{
		"role_ids": rb.Role.ID,
	}, bson.M{
		"$pull": bson.M{
			"role_ids": rb.Role.ID,
		},
	})
	if err != nil {
		zap.S().Errorw("mongo, error while deleting role from database")

		return errors.ErrInternalServerError()
	}

	return nil
}

type RoleMutationOptions struct {
	Actor *structures.User
}

type RoleEditOptions struct {
	Actor            *structures.User
	OriginalPosition int32
}
