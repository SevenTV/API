package mutate

import (
	"context"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
)

func (m *Mutate) SetMessageReadStates(ctx context.Context, mb *structures.MessageBuilder[bson.Raw], read bool, opt MessageReadStateOptions) (*MessageReadStateResponse, error) {
	if mb == nil {
		return nil, errors.ErrInternalIncompleteMutation()
	} else if mb.IsTainted() {
		return nil, errors.ErrMutateTaintedObject()
	}

	// Check permissions
	actor := opt.Actor
	if !opt.SkipPermissionCheck && actor == nil {
		return nil, errors.ErrUnauthorized()
	}

	// Find the readstates
	filter := bson.M{}
	if len(opt.Filter) > 0 {
		filter = opt.Filter
	}

	filter["message_id"] = mb.Message.ID
	cur, err := m.mongo.Collection(mongo.CollectionNameMessagesRead).Find(ctx, filter)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrUnknownMessage().SetDetail("Couldn't find any read states related to the message")
		}

		return nil, errors.ErrInternalServerError().SetDetail(err.Error())
	}

	errorList := []error{}
	w := []mongo.WriteModel{}

	for cur.Next(ctx) {
		rs := &structures.MessageRead{}
		if err := cur.Decode(rs); err != nil {
			continue
		}
		// Can actor do this?
		if !opt.SkipPermissionCheck && rs.RecipientID != actor.ID {
			switch rs.Kind {
			// Check for a mod request
			case structures.MessageKindModRequest:
				d, err := structures.ConvertMessage[structures.MessageDataModRequest](mb.Message)
				if err != nil {
					return nil, err
				}

				errf := errors.Fields{
					"message_id":       rs.MessageID.Hex(),
					"message_state_id": rs.ID.Hex(),
					"msg_kind":         rs.Kind,
					"target_kind":      d.Data.TargetKind,
				}

				if d.Data.TargetKind == structures.ObjectKindEmote && !actor.HasPermission(structures.RolePermissionEditAnyEmote) {
					errorList = append(errorList, errors.ErrInsufficientPrivilege().SetFields(errf))
					continue // target is emote but actor lacks "edit any emote" permission
				} else if d.Data.TargetKind == structures.ObjectKindEmoteSet && !actor.HasPermission(structures.RolePermissionEditAnyEmoteSet) {
					errorList = append(errorList, errors.ErrInsufficientPrivilege().SetFields(errf))
					continue // target is emote set but actor lacks "edit any emote set" permission
				} else if d.Data.TargetKind == structures.ObjectKindReport && !actor.HasPermission(structures.RolePermissionManageReports) {
					errorList = append(errorList, errors.ErrInsufficientPrivilege().SetFields(errf))
					continue // target is report but actor lacks "manage reports" permission
				}
			case structures.MessageKindInbox:
				errf := errors.Fields{
					"message_id":       rs.MessageID.Hex(),
					"message_state_id": rs.ID.Hex(),
					"msg_kind":         rs.Kind,
				}

				// Actor is not the recipient and is not privileged
				if !actor.HasPermission(structures.RolePermissionManageUsers) && actor.ID != rs.RecipientID {
					errorList = append(errorList, errors.ErrInsufficientPrivilege().SetFields(errf))
					continue
				}
			default:
				continue
			}
		}

		// Add as item to be written
		w = append(w, &mongo.UpdateOneModel{
			Filter: bson.M{"_id": rs.ID},
			Update: bson.M{"$set": bson.M{
				"read": read,
			}},
		})
	}

	updated := int64(0)

	if len(w) > 0 {
		result, err := m.mongo.Collection(mongo.CollectionNameMessagesRead).BulkWrite(ctx, w)
		if err != nil {
			return nil, errors.ErrInternalServerError().SetDetail(err.Error())
		}

		updated += result.ModifiedCount
	}

	mb.MarkAsTainted()

	return &MessageReadStateResponse{
		Updated: updated,
		Errors:  errorList,
	}, nil
}

type MessageReadStateOptions struct {
	Actor               *structures.User
	Filter              bson.M
	SkipPermissionCheck bool
}

type MessageReadStateResponse struct {
	Updated int64   `json:"changed"`
	Errors  []error `json:"errors"`
}
