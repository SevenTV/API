package query

import (
	"context"
	"strings"

	"github.com/seventv/api/data/model/modelgql"
	"github.com/seventv/api/data/query"
	"github.com/seventv/api/internal/api/gql/v3/auth"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ModRequests implements generated.QueryResolver
func (r *Resolver) ModRequests(ctx context.Context, afterIDArg *primitive.ObjectID, limitArg *int, wish *string, country *string) (*model.ModRequestMessageList, error) {
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
		match["_id"] = bson.M{"$lt": afterID}
	}

	if wish != nil {
		match["data.wish"] = *wish
	}

	if country != nil {
		match["data.actor_country_code"] = strings.ToUpper(*country)
	}

	limit := 50
	if limitArg != nil {
		limit = *limitArg
	}

	msgQuery := r.Ctx.Inst().Query.ModRequestMessages(ctx, query.ModRequestMessagesQueryOptions{
		Actor:  &actor,
		Filter: match,
		Limit:  limit,
		Sort:   bson.M{"_id": 1},
		Targets: map[structures.ObjectKind]bool{
			structures.ObjectKindEmote: true,
		},
	})

	messages, err := msgQuery.Items()
	if err != nil {
		errCode, _ := err.(errors.APIError)
		if errCode.Code() == errors.ErrNoItems().Code() {
			return &model.ModRequestMessageList{}, nil
		}

		return nil, err
	}

	result := make([]*model.ModRequestMessage, len(messages))

	for i, msg := range messages {
		if msg, err := structures.ConvertMessage[structures.MessageDataModRequest](msg); err == nil {
			result[i] = modelgql.ModRequestMessageModel(r.Ctx.Inst().Modelizer.ModRequestMessage(msg))
		}
	}

	return &model.ModRequestMessageList{
		Messages: result,
		Total:    int(msgQuery.Total()),
	}, nil
}
