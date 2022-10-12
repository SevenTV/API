package subscription

import (
	"context"

	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) CurrentUser(ctx context.Context, init *bool) (<-chan *model.UserPartial, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, nil
	}

	getUser := func() *model.UserPartial {
		user, err := r.Ctx.Inst().Loaders.UserByID().Load(actor.ID)
		if err != nil {
			return nil
		}

		return r.Ctx.Inst().Modelizer.User(user).PartialGQL()
	}

	ch := make(chan *model.UserPartial, 1)

	if init != nil && *init {
		user := getUser()
		if user != nil {
			ch <- user
		}
	}

	go func() {
		defer close(ch)

		sub := r.subscribe(ctx, "users", actor.ID)
		for range sub {
			user := getUser()
			if user != nil {
				ch <- user
			}
		}
	}()

	return ch, nil
}

func (r *Resolver) User(ctx context.Context, id primitive.ObjectID, init *bool) (<-chan *model.UserPartial, error) {
	getUser := func() *model.UserPartial {
		user, err := r.Ctx.Inst().Loaders.UserByID().Load(id)
		if err != nil {
			return nil
		}

		return r.Ctx.Inst().Modelizer.User(user).PartialGQL()
	}

	ch := make(chan *model.UserPartial, 1)

	if init != nil && *init {
		user := getUser()
		if user != nil {
			ch <- user
		}
	}

	go func() {
		defer close(ch)

		sub := r.subscribe(ctx, "users", id)
		for range sub {
			user := getUser()
			if user != nil {
				ch <- user
			}
		}
	}()

	return ch, nil
}
