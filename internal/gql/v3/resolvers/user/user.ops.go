package user

import (
	"context"

	"github.com/seventv/api/data/mutate"
	"github.com/seventv/api/internal/events"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/types"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type ResolverOps struct {
	types.Resolver
}

// Roles implements generated.UserOpsResolver
func (r *ResolverOps) Roles(ctx context.Context, obj *model.UserOps, roleID primitive.ObjectID, action model.ListItemAction) ([]primitive.ObjectID, error) {
	if action == model.ListItemActionUpdate {
		return nil, errors.ErrInvalidRequest().SetDetail("Cannot use UPDATE action for roles")
	}

	actor := auth.For(ctx)

	// Get target user
	user, err := r.Ctx.Inst().Query.Users(ctx, bson.M{
		"_id": obj.ID,
	}).First()
	if err != nil {
		if errors.Compare(err, errors.ErrNoItems()) {
			return nil, errors.ErrUnknownUser()
		}

		return nil, err
	}

	// Get role
	var role structures.Role

	roles, err := r.Ctx.Inst().Query.Roles(ctx, bson.M{
		"_id": roleID,
	})
	if err != nil || len(roles) == 0 {
		return nil, err
	} else {
		role = roles[0]
	}

	ub := structures.NewUserBuilder(user)

	if err = r.Ctx.Inst().Mutate.SetRole(ctx, ub, mutate.SetUserRoleOptions{
		Role:   role,
		Actor:  actor,
		Action: structures.ListItemAction(action),
	}); err != nil {
		return nil, err
	}

	events.Publish(r.Ctx, "users", obj.ID)

	if _, err := r.Ctx.Inst().CD.SyncUser(obj.ID); err != nil {
		r.Z().Errorw("failed to sync user with discord", "user", obj.ID, "err", err)
	}

	return ub.User.RoleIDs, nil
}

func (r *ResolverOps) Z() *zap.SugaredLogger {
	return zap.S().Named("user.ops")
}

func NewOps(r types.Resolver) generated.UserOpsResolver {
	return &ResolverOps{r}
}

func (r *ResolverOps) Connections(ctx context.Context, obj *model.UserOps, id string, d model.UserConnectionUpdate) ([]*model.UserConnection, error) {
	done := r.Ctx.Inst().Limiter.AwaitMutation(ctx)
	defer done()

	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	b := structures.NewUserBuilder(structures.DeletedUser)
	if err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).FindOne(ctx, bson.M{
		"_id": obj.ID,
	}).Decode(&b.User); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrUnknownUser()
		}

		return nil, errors.ErrInternalServerError().SetDetail(err.Error())
	}

	// Perform a mutation
	var err error

	// Unlink is mutually exclusive to all other mutation fields
	if d.Unlink != nil && *d.Unlink {
		if actor.ID != b.User.ID && !actor.HasPermission(structures.RolePermissionManageUsers) {
			return nil, errors.ErrInsufficientPrivilege()
		}

		if len(b.User.Connections) <= 1 {
			return nil, errors.ErrDontBeSilly().SetDetail("Cannot unlink the last connection, that would render your account inaccessible")
		}

		conn, ind := b.User.Connections.Get(id)
		if ind == -1 {
			return nil, errors.ErrUnknownUserConnection()
		}

		// If this is a discord connection, run a uer sync with the revoke param
		if conn.Platform == structures.UserConnectionPlatformDiscord {
			_, _ = r.Ctx.Inst().CD.RevokeUser(b.User.ID)
		}

		// Remove the connection and update the user
		if _, ind := b.RemoveConnection(conn.ID); ind >= 0 {
			// write to db
			if _, err = r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).UpdateOne(ctx, bson.M{
				"_id": obj.ID,
			}, b.Update); err != nil {
				if err == mongo.ErrNoDocuments {
					return nil, errors.ErrUnknownUser()
				}

				zap.S().Errorw("failed to update user", "error", err)

				return nil, errors.ErrInternalServerError()
			}
		}
	} else {
		if d.EmoteSetID != nil {
			conn, ind := b.User.Connections.Get(id)
			if ind == -1 {
				return nil, errors.ErrUnknownUserConnection()
			}

			newSet, _ := r.Ctx.Inst().Loaders.EmoteSetByID().Load(*d.EmoteSetID)
			oldSet, _ := r.Ctx.Inst().Loaders.EmoteSetByID().Load(conn.EmoteSetID)

			if err = r.Ctx.Inst().Mutate.SetUserConnectionActiveEmoteSet(ctx, b, mutate.SetUserActiveEmoteSet{
				NewSet:       newSet,
				OldSet:       oldSet,
				Platform:     structures.UserConnectionPlatformTwitch,
				Actor:        actor,
				ConnectionID: id,
			}); err != nil {
				return nil, err
			}
		}
	}

	if err != nil {
		return nil, err
	}

	result := r.Ctx.Inst().Modelizer.User(b.User).GQL()
	events.Publish(r.Ctx, "users", b.User.ID)

	return result.Connections, nil
}
