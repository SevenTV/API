package mutate

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/seventv/api/data/events"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const EMOTE_CLAIMANTS_MOST = 10

// Edit: edit the emote. Modify the EmoteBuilder beforehand!
//
// To account for editor permissions, the "editor_of" relation should be included in the actor's data
func (m *Mutate) EditEmote(ctx context.Context, eb *structures.EmoteBuilder, opt EmoteEditOptions) error {
	if eb == nil {
		return errors.ErrInternalIncompleteMutation()
	} else if eb.IsTainted() {
		return errors.ErrMutateTaintedObject()
	}

	actor := opt.Actor
	actorID := primitive.NilObjectID

	if !actor.ID.IsZero() {
		actorID = actor.ID
	}

	emote := &eb.Emote

	// Check actor's permission
	if !actor.ID.IsZero() {
		// User is not privileged
		if !actor.HasPermission(structures.RolePermissionEditAnyEmote) {
			if emote.OwnerID.IsZero() { // Deny when emote has no owner
				return errors.ErrInsufficientPrivilege()
			}

			// Check if actor is editor of the emote owner
			isPermittedEditor := false

			for _, ed := range actor.EditorOf {
				if ed.ID != emote.OwnerID {
					continue
				}
				// Allow if the actor has the "manage owned emotes" permission
				// as the editor of the emote owner
				if ed.HasPermission(structures.UserEditorPermissionManageOwnedEmotes) {
					isPermittedEditor = true
					break
				}
			}

			if emote.OwnerID != actor.ID && !isPermittedEditor { // Deny when not the owner or editor of the owner of the emote
				return errors.ErrInsufficientPrivilege()
			}
		}
	} else if !opt.SkipValidation {
		// if validation is not skipped then an Actor is mandatory
		return errors.ErrUnauthorized()
	}

	// Set up audit logs
	log := structures.NewAuditLogBuilder(structures.AuditLog{}).
		SetKind(structures.AuditLogKindUpdateEmote).
		SetActor(actorID).
		SetTargetKind(structures.ObjectKindEmote).
		SetTargetID(emote.ID)

	// Record field changes to emit an event about this mutation
	changeFields := []events.ChangeField{}

	if !opt.SkipValidation {
		init := eb.Initial()
		validator := eb.Emote.Validator()
		// Change: Name
		if init.Name != emote.Name {
			if err := validator.Name(); err != nil {
				return err
			}

			c := structures.AuditLogChange{
				Key:    "name",
				Format: structures.AuditLogChangeFormatSingleValue,
			}
			c.WriteSingleValues(init.Name, emote.Name)
			log.AddChanges(&c)

			changeFields = append(changeFields, events.ChangeField{
				Key:      "name",
				OldValue: init.Name,
				Value:    emote.Name,
			})
		}

		if init.OwnerID != emote.OwnerID {
			// Verify that the new emote exists
			if err := m.mongo.Collection(mongo.CollectionNameUsers).FindOne(ctx, bson.M{
				"_id": emote.OwnerID,
			}).Err(); err != nil {
				if err == mongo.ErrNoDocuments {
					return errors.ErrUnknownUser()
				}
			}

			// If the user is not privileged:
			// we will add the specified owner_id to list of claimants and send an inbox message
			switch init.OwnerID == actorID { // original owner is actor?
			case true: // yes: means emote owner is transferring away
				if !actor.HasPermission(structures.RolePermissionEditAnyEmote) {
					if utils.Contains(emote.State.Claimants, emote.OwnerID) { // error if target new owner is already a claimant
						return errors.ErrInsufficientPrivilege().SetDetail("Target user was already requested to claim ownership of this emote")
					}

					if len(emote.State.Claimants) > EMOTE_CLAIMANTS_MOST {
						return errors.ErrInvalidRequest().SetDetail("Too Many Claimants (%d)", EMOTE_CLAIMANTS_MOST)
					}

					// Add to claimants
					eb.Update.AddToSet("state.claimants", emote.OwnerID)

					// Send a message to the claimant's inbox
					mb := structures.NewMessageBuilder(structures.Message[structures.MessageDataInbox]{}).
						SetKind(structures.MessageKindInbox).
						SetAuthorID(actorID).
						SetTimestamp(time.Now()).
						SetData(structures.MessageDataInbox{
							Subject: "inbox.generic.emote_ownership_claim_request.subject",
							Content: "inbox.generic.emote_ownership_claim_request.content",
							Locale:  true,
							Placeholders: map[string]string{
								"OWNER_DISPLAY_NAME":  utils.Ternary(emote.Owner.DisplayName != "", emote.Owner.DisplayName, emote.Owner.Username),
								"EMOTE_VERSION_COUNT": strconv.Itoa(len(emote.Versions)),
								"EMOTE_NAME":          emote.Name,
							},
						})
					if err := m.SendInboxMessage(ctx, mb, SendInboxMessageOptions{
						Actor:                &actor,
						Recipients:           []primitive.ObjectID{emote.OwnerID},
						ConsiderBlockedUsers: true,
					}); err != nil {
						return err
					}

					// Undo owner update
					eb.Update.UndoSet("owner_id")

					emote.OwnerID = init.OwnerID
				}
			case false: // no: a user wants to claim ownership
				// Check if actor is allowed to do that
				if !actor.HasPermission(structures.RolePermissionEditAnyEmote) {
					if emote.OwnerID != actorID { //
						return errors.ErrInsufficientPrivilege().SetDetail("You are not permitted to change this emote's owner")
					}

					if !utils.Contains(emote.State.Claimants, emote.OwnerID) {
						return errors.ErrInsufficientPrivilege().SetDetail("You are not allowed to claim ownership of this emote")
					}
				}
				// At this point the actor has successfully claimed ownership of the emote and we clear the list of claimants
				eb.Update.Set("state.claimants", []primitive.ObjectID{})

				// Write as audit change
				c := structures.AuditLogChange{
					Key:    "owner_id",
					Format: structures.AuditLogChangeFormatSingleValue,
				}
				c.WriteSingleValues(init.OwnerID, emote.OwnerID)
				log.AddChanges(&c)

				changeFields = append(changeFields, events.ChangeField{
					Key:      "owner_id",
					OldValue: init.OwnerID,
					Value:    emote.OwnerID,
				})
			}
		}

		if utils.DifferentArray(init.Tags, emote.Tags) {
			eb.SetTags(emote.Tags, true)

			c := structures.AuditLogChange{
				Key:    "tags",
				Format: structures.AuditLogChangeFormatSingleValue,
			}

			c.WriteSingleValues(init.Tags, emote.Tags)
			log.AddChanges(&c)

			changeFields = append(changeFields, events.ChangeField{
				Key:      "tags",
				OldValue: init.Tags,
				Value:    emote.Tags,
			})
		}

		if init.Flags != emote.Flags {
			f := emote.Flags

			// Validate privileged flags
			if !actor.HasPermission(structures.RolePermissionEditAnyEmote) {
				privilegedBits := []structures.EmoteFlag{
					structures.EmoteFlagsContentSexual,
					structures.EmoteFlagsContentEpilepsy,
					structures.EmoteFlagsContentEdgy,
					structures.EmoteFlagsContentTwitchDisallowed,
				}
				for _, flag := range privilegedBits {
					if f.Has(flag) {
						return errors.ErrInsufficientPrivilege().SetDetail("Not allowed to modify flag %s", flag.String())
					}
				}
			}

			c := structures.AuditLogChange{
				Key:    "flags",
				Format: structures.AuditLogChangeFormatSingleValue,
			}
			c.WriteSingleValues(init.Flags, emote.Flags)
			log.AddChanges(&c)

			changeFields = append(changeFields, events.ChangeField{
				Key:      "flags",
				OldValue: init.Flags,
				Value:    emote.Flags,
			})
		}

		// Change versions
		oldVersions := eb.InitialVersions()
		oldVersionMap := make(map[primitive.ObjectID]structures.EmoteVersion)

		for _, v := range oldVersions {
			oldVersionMap[v.ID] = v
		}

		for i, ver := range emote.Versions {
			oldVer := oldVersionMap[ver.ID]
			if oldVer.ID.IsZero() {
				continue // cannot update version that didn't exist
			}

			o := make(map[string]any)
			n := make(map[string]any)
			c := structures.AuditLogChange{
				Key:    "versions",
				Format: structures.AuditLogChangeFormatSingleValue,
			}

			localChangeFields := []events.ChangeField{}

			// Update: listed
			changeCount := 0

			if ver.State.Listed != oldVer.State.Listed {
				if !actor.HasPermission(structures.RolePermissionEditAnyEmote) {
					return errors.ErrInsufficientPrivilege().SetDetail("Not allowed to modify listed state of version %s", strconv.Itoa(i))
				}

				n["listed"] = ver.State.Listed
				o["listed"] = oldVer.State.Listed
				changeCount++

				localChangeFields = append(localChangeFields, events.ChangeField{
					Key:      "listed",
					OldValue: oldVer.State.Listed,
					Value:    ver.State.Listed,
				})
			}

			if ver.Name != "" && ver.Name != oldVer.Name {
				if err := ver.Validator().Name(); err != nil {
					return err
				}

				n["name"] = ver.Name
				o["name"] = oldVer.Name
				changeCount++

				localChangeFields = append(localChangeFields, events.ChangeField{
					Key:      "name",
					OldValue: oldVer.Name,
					Value:    ver.Name,
				})
			}

			if ver.Description != "" && ver.Description != oldVer.Description {
				if err := ver.Validator().Description(); err != nil {
					return err
				}

				n["description"] = ver.Description
				o["description"] = oldVer.Description
				changeCount++

				localChangeFields = append(localChangeFields, events.ChangeField{
					Key:      "description",
					OldValue: oldVer.Description,
					Value:    ver.Description,
				})
			}

			if ver.State.AllowPersonal != oldVer.State.AllowPersonal {
				n["allow_personal"] = ver.State.AllowPersonal
				o["allow_personal"] = oldVer.State.AllowPersonal
				changeCount++
			}

			if changeCount > 0 {
				c.WriteArrayUpdated(structures.AuditLogChangeSingleValue{
					New:      n,
					Old:      o,
					Position: int32(i),
				})
				log.AddChanges(&c)

				// Nested field changes
				changeFields = append(changeFields, events.ChangeField{
					Key:    "versions",
					Nested: true,
					Index:  utils.PointerOf(int32(i)),
					Value:  localChangeFields,
				})
			}
		}
	}

	// Update the emote
	if len(eb.Update) > 0 {
		if err := m.mongo.Collection(mongo.CollectionNameEmotes).FindOneAndUpdate(
			ctx,
			bson.M{"_id": emote.ID},
			eb.Update,
			options.FindOneAndUpdate().SetReturnDocument(options.After),
		).Decode(emote); err != nil {
			zap.S().Errorw("mongo, couldn't edit emote",
				"error", err,
			)

			return errors.ErrInternalServerError().SetDetail(err.Error())
		}

		// Write audit log entry
		if len(log.AuditLog.Changes) > 0 {
			go func() {
				if _, err := m.mongo.Collection(mongo.CollectionNameAuditLogs).InsertOne(ctx, log.AuditLog); err != nil {
					zap.S().Errorw("failed to write audit log",
						"error", err,
					)
				}
			}()
		}

		if len(changeFields) > 0 {
			for _, ver := range emote.Versions {
				go func(ver structures.EmoteVersion) {
					// Emit to the Event API
					m.events.Dispatch(ctx, events.EventTypeUpdateEmote, events.ChangeMap{
						ID:      ver.ID,
						Kind:    structures.ObjectKindEmote,
						Actor:   m.modelizer.User(actor).ToPartial(),
						Updated: changeFields,
					}, events.EventCondition{
						"object_id": ver.ID.Hex(),
					})
				}(ver)
			}

			cdContent := strings.Builder{}

			for _, cf := range changeFields {
				cdContent.WriteString(cf.String())

				cdContent.WriteString(" ")
			}

			_, _ = m.cd.SendMessage("mod_actor_tracker", discordgo.MessageSend{
				Content: fmt.Sprintf("**[edit]** **[%s]** ✏️ [%s](%s) %s", actor.Username, eb.Emote.Name, eb.Emote.WebURL(m.id.Web), cdContent.String()),
			}, true)
		}
	}

	eb.MarkAsTainted()

	return nil
}

type EmoteEditOptions struct {
	Actor          structures.User
	SkipValidation bool
}
