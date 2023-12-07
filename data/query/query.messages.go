package query

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/aggregations"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	return q.Messages(ctx, bson.M{
		"recipient_id": user.ID,
		"kind":         structures.MessageKindInbox,
	}, MessageQueryOptions{
		Actor: actor,
		Limit: opt.Limit,
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
		opt.Sort = bson.D{{Key: "_id", Value: -1}}
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

	h := sha256.New()

	fb, err := json.Marshal(filter)
	if err == nil {
		h.Write(fb)
	}

	rKey := q.redis.ComposeKey("api", "messages", hex.EncodeToString(h.Sum(nil)), "count")

	go func() {
		defer wg.Done()

		count, err := q.redis.Get(ctx, rKey)
		if err != redis.Nil {
			n, _ := strconv.ParseInt(count, 10, 64)

			qr.setTotal(n)
			return
		}

		cur, err := q.mongo.Collection(mongo.CollectionNameMessagesRead).Aggregate(ctx, aggregations.Combine(
			matcherPipeline,
			func() mongo.Pipeline {
				if len(opt.MessageFilter) == 0 {
					return mongo.Pipeline{}
				}

				return mongo.Pipeline{
					{{
						Key: "$match",
						Value: bson.M{
							"message": bson.M{
								"$elemMatch": opt.MessageFilter,
							},
						},
					}}}
			}(),
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

		q.redis.SetEX(ctx, rKey, v.Count, time.Minute*5)
	}()

	cur, err := q.mongo.Collection(mongo.CollectionNameMessagesRead).Aggregate(ctx, aggregations.Combine(
		matcherPipeline,
		mongo.Pipeline{
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
								"weight":    "$weight",
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
		mongo.Pipeline{
			{{Key: "$limit", Value: opt.Limit}},
		},
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
	Sort                bson.D
	Limit               int
	SkipPermissionCheck bool
}

type MessageQueryOptions struct {
	Actor            *structures.User
	Limit            int
	UnreadOnly       bool
	MessageFilter    bson.M
	FilterRecipients []primitive.ObjectID
	Sort             bson.D
}
