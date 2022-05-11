package mutation

import (
	"context"
	"time"

	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/Common/structures/v3/mutations"
	"github.com/SevenTV/Common/structures/v3/query"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) CreateBan(ctx context.Context, victimID primitive.ObjectID, reason string, effects int, expireAtArg *time.Time, anonymousArg *bool) (*model.Ban, error) {
	// Get the actor uszer²
	actor := auth.For(ctx)
	if actor == nil {
		return nil, errors.ErrUnauthorized()
	}

	// When the ban expires
	expireAt := time.Now().AddDate(0, 0, 3)
	if expireAtArg != nil {
		expireAt = *expireAtArg
	}

	// Fetch the victim user
	var victim *structures.User
	if users, _, err := r.Ctx.Inst().Query.SearchUsers(ctx, bson.M{"_id": victimID}, query.UserSearchOptions{Page: 1, Limit: 1}); err == nil && len(users) > 0 {
		victim = &users[0]
	} else {
		if len(users) == 0 {
			return nil, errors.ErrUnknownUser().SetDetail("Victim not found")
		}
		return nil, err
	}

	// Create the ban
	bb := structures.NewBanBuilder(structures.Ban{}).
		SetActorID(actor.ID).
		SetVictimID(victim.ID).
		SetReason(reason).
		SetExpireAt(expireAt).
		SetEffects(structures.BanEffect(effects))
	if err := r.Ctx.Inst().Mutate.CreateBan(ctx, bb, mutations.CreateBanOptions{
		Actor:  actor,
		Victim: victim,
	}); err != nil {
		return nil, err
	}

	return helpers.BanStructureToModel(r.Ctx, bb.Ban), nil
}

func (r *Resolver) EditBan(ctx context.Context, banID primitive.ObjectID, reason *string, effects *int, expireAt *string) (*model.Ban, error) {
	// TODO
	return nil, nil
}
