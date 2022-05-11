package mutation

import (
	"context"
	"time"

	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/Common/structures/v3/mutations"
	"github.com/SevenTV/Common/structures/v3/query"
	"github.com/seventv/api/gql/v2/gen/model"
	"github.com/seventv/api/gql/v3/auth"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BanUser implements generated.MutationResolver
func (r *Resolver) BanUser(ctx context.Context, victimIDArg string, expireAtArg *string, reasonArg *string) (*model.Response, error) {
	actor := auth.For(ctx)
	if actor == nil {
		return nil, errors.ErrUnauthorized()
	}

	// Parse expiry time
	var err error
	expireAt := time.Now().AddDate(0, 0, 3)
	if expireAtArg != nil {
		expireAt, err = time.Parse(time.RFC3339, *expireAtArg)
		if err != nil {
			return nil, errors.ErrInvalidRequest().SetDetail("Invalid expire date: %s", err.Error())
		}
	}

	// Reason
	reason := "No reason"
	if reasonArg != nil {
		reason = *reasonArg
	}

	// Fetch the victim user
	var victim structures.User
	victimID, err := primitive.ObjectIDFromHex(victimIDArg)
	if err != nil {
		return nil, errors.ErrBadObjectID()
	}
	if user, err := r.Ctx.Inst().Query.Users(ctx, bson.M{"_id": victimID}).First(); err == nil {
		victim = user
	} else {
		return nil, err
	}

	// Create the ban
	bb := structures.NewBanBuilder(structures.Ban{}).
		SetActorID(actor.ID).
		SetVictimID(victim.ID).
		SetReason(reason).
		SetExpireAt(expireAt).
		SetEffects(structures.BanEffect(structures.BanEffectMemoryHole | structures.BanEffectNoAuth | structures.BanEffectNoPermissions))
	if err = r.Ctx.Inst().Mutate.CreateBan(ctx, bb, mutations.CreateBanOptions{
		Actor:  actor,
		Victim: &victim,
	}); err != nil {
		return nil, err
	}

	return &model.Response{
		Status:  200,
		Ok:      true,
		Message: "success",
	}, nil
}

// UnbanUser implements generated.MutationResolver
func (r *Resolver) UnbanUser(ctx context.Context, victimIDArg string, reason *string) (*model.Response, error) {
	actor := auth.For(ctx)
	if actor == nil {
		return nil, errors.ErrUnauthorized()
	}

	// Parse victim ID
	victimID, err := primitive.ObjectIDFromHex(victimIDArg)
	if err != nil {
		return nil, errors.ErrBadObjectID()
	}
	// Find victim
	users := r.Ctx.Inst().Query.Users(ctx, bson.M{"_id": victimID})
	if users.Error() != nil {
		return nil, errors.ErrUnknownUser()
	}

	// Find bans on victim
	// because this is v2, we don't really understand
	// the concept of multiple bans with varying effects
	// so unbanning from v2 will cancel out *all* active bans
	bans, err := r.Ctx.Inst().Query.Bans(ctx, query.BanQueryOptions{
		Filter: bson.M{"victim_id": victimID},
	})
	if err != nil {
		return nil, err
	}

	for _, ban := range bans.All {
		bb := structures.NewBanBuilder(ban)
		// Change expire date to current date
		// (equivalent of setting active: false in v2)
		bb.SetExpireAt(time.Now())
		if err = r.Ctx.Inst().Mutate.EditBan(ctx, bb, mutations.EditBanOptions{
			Actor: actor,
		}); err != nil {
			logrus.WithError(err).Error("failed to perform v2 unban user operation")
		}
	}
	return &model.Response{
		Status:  200,
		Ok:      true,
		Message: "success",
	}, nil
}
