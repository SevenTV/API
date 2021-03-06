package query

import (
	"context"

	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/query"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ModRequests implements generated.QueryResolver
func (r *Resolver) ModRequests(ctx context.Context, afterIDArg *primitive.ObjectID) ([]*model.ModRequestMessage, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	afterID := primitive.NilObjectID
	if afterIDArg != nil {
		afterID = *afterIDArg
	}

	match := bson.M{}
	if !afterID.IsZero() {
		match["message_id"] = bson.M{"$gt": afterID}
	}

	messages, err := r.Ctx.Inst().Query.ModRequestMessages(ctx, query.ModRequestMessagesQueryOptions{
		Actor:  &actor,
		Filter: match,
		Targets: map[structures.ObjectKind]bool{
			structures.ObjectKindEmote: true,
		},
	}).Items()
	if err != nil {
		errCode, _ := err.(errors.APIError)
		if errCode.Code() == errors.ErrNoItems().Code() {
			return []*model.ModRequestMessage{}, nil
		}

		return nil, err
	}

	result := make([]*model.ModRequestMessage, len(messages))

	for i, msg := range messages {
		if msg, err := structures.ConvertMessage[structures.MessageDataModRequest](msg); err == nil {
			result[i] = helpers.MessageStructureToModRequestModel(msg, r.Ctx.Config().CdnURL)
		}
	}

	return result, nil
}
