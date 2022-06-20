package mutation

import (
	"context"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/mutations"
	"github.com/seventv/common/structures/v3/query"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const MESSAGE_RECIPIENTS_MOST = 20

func (r *Resolver) ReadMessages(ctx context.Context, messageIds []primitive.ObjectID, read bool) (int, error) {
	actor := auth.For(ctx)
	if actor == nil {
		return 0, errors.ErrUnauthorized()
	}

	// Fetch messages
	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameMessages).Find(ctx, bson.M{
		"_id": bson.M{"$in": messageIds},
	})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, errors.ErrUnknownMessage().SetDetail("No messages found")
		}
		return 0, errors.ErrInternalServerError().SetDetail(err.Error())
	}

	// Mutate messages
	messages := []structures.Message[bson.Raw]{}
	if err := cur.All(ctx, &messages); err != nil {
		return 0, errors.ErrInternalServerError().SetDetail(err.Error())
	}

	updated := 0
	for _, msg := range messages {
		result, err := r.Ctx.Inst().Mutate.SetMessageReadStates(ctx, structures.NewMessageBuilder(msg), read, mutations.MessageReadStateOptions{
			Actor:               actor,
			SkipPermissionCheck: false,
		})
		if result != nil {
			for _, er := range result.Errors {
				graphql.AddError(ctx, er)
			}
		}
		if err != nil {
			return 0, err
		}
		updated += int(result.Updated)
	}
	return updated, nil
}

func (r *Resolver) SendInboxMessage(ctx context.Context, recipientsArg []primitive.ObjectID, subject string, content string, importantArg *bool, anonArg *bool) (*model.InboxMessage, error) {
	actor := auth.For(ctx)
	if actor == nil {
		return nil, errors.ErrUnauthorized()
	}

	// Verify maximum amount of recipients
	if len(recipientsArg) > MESSAGE_RECIPIENTS_MOST {
		return nil, errors.ErrInvalidRequest().
			SetDetail("Too Many Recipients (got %d, but the most I'll accept is %d)", len(recipientsArg), MESSAGE_RECIPIENTS_MOST).
			SetFields(errors.Fields{
				"RECIPIENT_AMOUNT": len(recipientsArg),
				"RECIPIENTS_MOST":  MESSAGE_RECIPIENTS_MOST,
			})
	}

	// Actor is allowed to be annonymous
	anonymous := false
	if anonArg != nil && *anonArg {
		if !actor.HasPermission(structures.RolePermissionBypassPrivacy) {
			return nil, errors.ErrInsufficientPrivilege().
				SetDetail("You are not permitted to send messages anonnymously").
				SetFields(errors.Fields{"MISSING_PERMISSION": "BYPASS_PRIVACY"})
		}
		anonymous = true
	}

	// Mark message as important?
	important := false
	if importantArg != nil && *importantArg {
		if !actor.HasPermission(structures.RolePermissionManageUsers) || !actor.HasPermission(structures.RolePermissionManageNews) {
			return nil, errors.ErrInsufficientPrivilege().
				SetDetail("You are not permitted to send messages marked as important").
				SetFields(errors.Fields{"MISSING_PERMISSION_ONE_OF": []string{"MANAGE_USERS", "MANAGE_NEWS"}})
		}
		important = true
	}

	// Fetch recipients
	recipients := r.Ctx.Inst().Query.Users(ctx, bson.M{"_id": bson.M{"$in": recipientsArg}})
	if recipients.Error() != nil {
		return nil, recipients.Error()
	}

	//
	mb := structures.NewMessageBuilder(structures.Message[structures.MessageDataInbox]{}).
		SetKind(structures.MessageKindInbox).
		SetAuthorID(actor.ID).
		SetTimestamp(time.Now()).
		SetAnonymous(anonymous).
		SetData(structures.MessageDataInbox{
			Subject:   subject,
			Content:   content,
			Important: important,
		})
	if err := r.Ctx.Inst().Mutate.SendInboxMessage(ctx, mb, mutations.SendInboxMessageOptions{
		Actor:                actor,
		Recipients:           recipientsArg,
		ConsiderBlockedUsers: !actor.HasPermission(structures.RolePermissionBypassPrivacy),
	}); err != nil {
		return nil, err
	}

	msg, err := r.Ctx.Inst().Query.Messages(ctx, bson.M{"_id": mb.Message.ID}, query.MessageQueryOptions{
		Actor:        actor,
		Limit:        1,
		ReturnUnread: true,
	}).First()
	if err != nil {
		return nil, err
	}

	inb, err := structures.ConvertMessage[structures.MessageDataInbox](msg)
	if err != nil {
		return nil, err
	}
	return helpers.MessageStructureToInboxModel(inb, r.Ctx.Config().CdnURL), nil
}
