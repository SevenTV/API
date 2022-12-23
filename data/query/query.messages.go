package query

import (
	"context"
	"sync"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/aggregations"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func (q *Query) InboxMessages(ctx context.Context, opt InboxMessagesQueryOptions) *QueryResult[structures.Message[bson.Raw]] {
	qr := &QueryResult[structures.Message[bson.Raw]]{}
	actor := opt.Actor
	user := opt.User

	if user == nil {
		return qr.setError(errors.ErrInternalServerError().SetDetail("no user passed to Inbox query"))
	}

	if !opt.SkipPermissionCheck {
		if actor == nil {
			return qr.setError(errors.ErrUnauthorized())
		}

		// Actor is not the target user
		if actor.ID != user.ID {
			ed, ok, _ := user.GetEditor(actor.ID)
			// Actor is not editor of target user
			if !ok {
				return qr.setError(errors.ErrInsufficientPrivilege().SetDetail("You are not an editor of this user"))
			}
			// Actor is an editor, but does not have the permission to do this
			if !ed.HasPermission(structures.UserEditorPermissionViewMessages) {
				return qr.setError(errors.ErrInsufficientPrivilege().SetDetail("You are not allowed to view the messages of this user"))
			}
		}
	}

	// Fetch message read states where target user is recipient
	cur, err := q.mongo.Collection(mongo.CollectionNameMessagesRead).Find(ctx, bson.M{
		"recipient_id": user.ID,
		"kind":         structures.MessageKindInbox,
	}, options.Find().SetProjection(bson.M{"message_id": 1}))
	if err != nil {
		return qr.setError(errors.ErrInternalServerError().SetDetail(err.Error()))
	}

	messageIDs := []primitive.ObjectID{}

	for cur.Next(ctx) {
		msg := &structures.MessageRead{}
		if err = cur.Decode(msg); err != nil {
			continue
		}

		messageIDs = append(messageIDs, msg.MessageID)
	}

	and := bson.A{bson.M{"_id": bson.M{"$in": messageIDs}}}
	if !opt.AfterID.IsZero() {
		and = append(and, bson.M{"_id": bson.M{"$gt": opt.AfterID}})
	}

	return q.Messages(ctx, bson.M{"$and": and}, MessageQueryOptions{
		Actor:            actor,
		Limit:            opt.Limit,
		FilterRecipients: []primitive.ObjectID{user.ID},
	})
}

func (q *Query) ModRequestMessages(ctx context.Context, opt ModRequestMessagesQueryOptions) *QueryResult[structures.Message[bson.Raw]] {
	qr := &QueryResult[structures.Message[bson.Raw]]{}
	actor := opt.Actor
	targets := opt.Targets

	if !opt.SkipPermissionCheck {
		if actor == nil {
			return qr.setError(errors.ErrUnauthorized())
		}

		// check permissions for targets
		if !actor.HasPermission(structures.RolePermissionEditAnyEmote) {
			targets[structures.ObjectKindEmote] = false
		}

		if !actor.HasPermission(structures.RolePermissionEditAnyEmoteSet) {
			targets[structures.ObjectKindEmoteSet] = false
		}

		if !actor.HasPermission(structures.RolePermissionManageReports) {
			targets[structures.ObjectKindReport] = false
		}
	}

	return q.Messages(ctx, bson.M{
		"kind": structures.MessageKindModRequest,
	}, MessageQueryOptions{
		UnreadOnly:    true,
		MessageFilter: opt.Filter,
		Actor:         actor,
		Sort:          opt.Sort,
		Limit:         opt.Limit,
	})
}

func (q *Query) Messages(ctx context.Context, filter bson.M, opt MessageQueryOptions) *QueryResult[structures.Message[bson.Raw]] {
	qr := &QueryResult[structures.Message[bson.Raw]]{}

	if opt.Sort == nil {
		opt.Sort = bson.M{"_id": -1}
	}

	if opt.UnreadOnly {
		filter["read"] = false
	}

	matcherPipeline := mongo.Pipeline{
		{{Key: "$sort", Value: opt.Sort}},
		{{Key: "$match", Value: filter}},
		{{
			Key: "$lookup",
			Value: mongo.Lookup{
				From:         mongo.CollectionNameMessages,
				LocalField:   "message_id",
				ForeignField: "_id",
				As:           "message",
			},
		}},
		{{
			Key: "$match",
			Value: bson.M{
				"message": bson.M{"$size": 1},
			},
		}},
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()

		cur, err := q.mongo.Collection(mongo.CollectionNameMessagesRead).Aggregate(ctx, aggregations.Combine(
			matcherPipeline,
			mongo.Pipeline{{{Key: "$count", Value: "count"}}},
		))
		if err != nil {
			zap.S().Errorw("failed to count total messages", "error", err)

			return
		}

		v := struct {
			Count int64 `bson:"count"`
		}{}

		if cur.Next(ctx) {
			if err = cur.Decode(&v); err != nil {
				zap.S().Errorw("failed to decode total messages", "error", err)
			}
		}

		qr.setTotal(v.Count)
	}()

	cur, err := q.mongo.Collection(mongo.CollectionNameMessagesRead).Aggregate(ctx, aggregations.Combine(
		matcherPipeline,
		mongo.Pipeline{
			{{Key: "$limit", Value: opt.Limit}},
			{{
				Key: "$set",
				Value: bson.M{
					"message": bson.M{
						"$arrayElemAt": bson.A{"$message", 0},
					},
				},
			}},
			{{
				Key: "$replaceRoot",
				Value: bson.M{
					"newRoot": bson.M{
						"$mergeObjects": bson.A{
							"$message",
							bson.M{
								"timestamp": "$timestamp",
								"read":      "$read",
							},
						},
					},
				},
			}},
		},
		func() mongo.Pipeline {
			if len(opt.MessageFilter) == 0 {
				return mongo.Pipeline{}
			}

			return mongo.Pipeline{
				{{
					Key:   "$match",
					Value: opt.MessageFilter,
				}}}
		}(),
	))
	if err != nil {
		zap.S().Errorw("failed to create messages aggregation", "error", err)

		return qr.setError(errors.ErrInternalServerError())
	}

	v := []structures.Message[bson.Raw]{}

	if err = cur.All(ctx, &v); err != nil {
		zap.S().Errorw("failed to decode messages", "error", err)

		return qr.setError(errors.ErrInternalServerError())
	}

	wg.Wait()

	return qr.setItems(v)
}

type InboxMessagesQueryOptions struct {
	Actor               *structures.User
	User                *structures.User // The user to fetch inbox messagesq from
	Limit               int
	AfterID             primitive.ObjectID
	SkipPermissionCheck bool
}

type ModRequestMessagesQueryOptions struct {
	Actor               *structures.User
	Targets             map[structures.ObjectKind]bool
	TargetIDs           []primitive.ObjectID
	Filter              bson.M
	Sort                bson.M
	Limit               int
	SkipPermissionCheck bool
}

type MessageQueryOptions struct {
	Actor            *structures.User
	Limit            int
	UnreadOnly       bool
	MessageFilter    bson.M
	FilterRecipients []primitive.ObjectID
	Sort             bson.M
}
