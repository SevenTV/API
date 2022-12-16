package query

import (
	"context"

	"github.com/seventv/api/data/model/modelgql"
	"github.com/seventv/api/data/query"
	"github.com/seventv/api/internal/api/gql/v3/auth"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const INBOX_QUERY_LIMIT_MOST = 1000

func (r *Resolver) Inbox(ctx context.Context, userID primitive.ObjectID, afterIDArg *primitive.ObjectID, limitArg *int) ([]*model.InboxMessage, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	// Pagination
	afterID := primitive.NilObjectID
	if afterIDArg != nil {
		afterID = *afterIDArg
	}

	limit := 100
	if limitArg != nil {
		limit = *limitArg
		if limit > INBOX_QUERY_LIMIT_MOST {
			limit = INBOX_QUERY_LIMIT_MOST
		}
	}

	// Fetch target user
	user, err := r.Ctx.Inst().Query.Users(ctx, bson.M{"_id": userID}).First()
	if err != nil {
		return nil, err
	}

	messages, err := r.Ctx.Inst().Query.InboxMessages(ctx, query.InboxMessagesQueryOptions{
		Actor:               &actor,
		User:                &user,
		Limit:               limit,
		AfterID:             afterID,
		SkipPermissionCheck: false,
	}).Items()
	if err != nil {
		return nil, err
	}

	result := make([]*model.InboxMessage, len(messages))

	for i, msg := range messages {
		if msg, err := structures.ConvertMessage[structures.MessageDataInbox](msg); err == nil {
			result[i] = modelgql.InboxMessageModel(r.Ctx.Inst().Modelizer.InboxMessage(msg))
		}
	}

	return result, nil
}
