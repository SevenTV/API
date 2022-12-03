package mutation

import (
	"context"
	"time"

	"github.com/seventv/api/data/mutate"
	"github.com/seventv/api/internal/api/gql/v3/auth"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/api/internal/api/gql/v3/helpers"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) CreateBan(ctx context.Context, victimID primitive.ObjectID, reason string, effects int, expireAtArg *time.Time, anonymousArg *bool) (*model.Ban, error) {
	// Get the actor uszer²
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	// When the ban expires
	expireAt := time.Now().AddDate(0, 0, 3)
	if expireAtArg != nil {
		expireAt = *expireAtArg
	}

	// Fetch the victim user
	victim, err := r.Ctx.Inst().Query.Users(ctx, bson.M{"_id": victimID}).First()
	if err != nil {
		if victim.ID.IsZero() {
			return nil, errors.ErrUnknownUser()
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
	if err := r.Ctx.Inst().Mutate.CreateBan(ctx, bb, mutate.CreateBanOptions{
		Actor:  &actor,
		Victim: &victim,
	}); err != nil {
		return nil, err
	}

	return helpers.BanStructureToModel(bb.Ban), nil
}

func (r *Resolver) EditBan(ctx context.Context, banID primitive.ObjectID, reason *string, effects *int, expireAt *string) (*model.Ban, error) {
	actor := auth.For(ctx)

	ban := structures.Ban{}
	if err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameBans).FindOne(ctx, bson.M{
		"_id": banID,
	}).Decode(&ban); err != nil {
		return nil, err
	}

	bb := structures.NewBanBuilder(ban)

	if reason != nil {
		bb.SetReason(*reason)
	}

	if effects != nil {
		bb.SetEffects(structures.BanEffect(*effects))
	}

	if expireAt != nil {
		at, err := time.Parse(time.RFC3339, *expireAt)
		if err != nil {
			return nil, errors.ErrInvalidRequest().SetDetail("Unable to parse time: %s", err.Error())
		}

		bb.SetExpireAt(at)
	}

	if err := r.Ctx.Inst().Mutate.EditBan(ctx, bb, mutate.EditBanOptions{
		Actor: &actor,
	}); err != nil {
		return nil, err
	}

	return nil, nil
}
