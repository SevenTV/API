package mutate

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/data/events"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/aggregations"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const (
	EMOTE_SET_PERSONAL_MIN_EMOTE_LIFETIME = time.Hour
)

// SetEmote: enable, edit or disable active emotes in the set
func (m *Mutate) EditEmotesInSet(ctx context.Context, esb *structures.EmoteSetBuilder, opt EmoteSetMutationSetEmoteOptions) error {
	if esb == nil {
		return errors.ErrInternalIncompleteMutation()
	} else if esb.IsTainted() {
		return errors.ErrMutateTaintedObject()
	}

	if len(opt.Emotes) == 0 {
		return errors.ErrMissingRequiredField().SetDetail("EmoteIDs")
	}

	// Can actor do this?
	actor := opt.Actor
	if actor.ID.IsZero() || !actor.HasPermission(structures.RolePermissionEditEmoteSet) {
		return errors.ErrInsufficientPrivilege().SetFields(errors.Fields{"MISSING_PERMISSION": "EDIT_EMOTE_SET"})
	}

	// Get relevant data
	targetEmoteIDs := []primitive.ObjectID{}
	targetEmoteMap := map[primitive.ObjectID]EmoteSetMutationSetEmoteItem{}
	set := esb.EmoteSet
	{
		// Find emote set owner
		if set.Owner == nil {
			set.Owner = &structures.User{}
			cur, err := m.mongo.Collection(mongo.CollectionNameUsers).Aggregate(ctx, append(mongo.Pipeline{
				{{Key: "$match", Value: bson.M{"_id": set.OwnerID}}},
			}, aggregations.UserRelationEditors...))
			cur.Next(ctx)
			if err = multierror.Append(err, cur.Decode(set.Owner), cur.Close(ctx)).ErrorOrNil(); err != nil {
				if err == mongo.ErrNoDocuments {
					return errors.ErrUnknownUser().SetDetail("emote set owner")
				}
				return err
			}
		}

		// Fetch set emotes
		if len(set.Emotes) == 0 {
			cur, err := m.mongo.Collection(mongo.CollectionNameEmoteSets).Aggregate(ctx, append(mongo.Pipeline{
				// Match only the target set
				{{Key: "$match", Value: bson.M{"_id": set.ID}}},
			}, aggregations.EmoteSetRelationActiveEmotes...))
			if err = multierror.Append(err, cur.All(ctx, &set.Emotes)).ErrorOrNil(); err != nil {
				return err
			}
		}

		// Fetch target emotes
		for _, e := range opt.Emotes {
			targetEmoteIDs = append(targetEmoteIDs, e.ID)
			targetEmoteMap[e.ID] = e
		}
		targetEmotes := []*structures.Emote{}
		cur, err := m.mongo.Collection(mongo.CollectionNameEmotes).Aggregate(ctx, append(mongo.Pipeline{
			{{Key: "$match", Value: bson.M{"versions.id": bson.M{"$in": targetEmoteIDs}}}},
		}, aggregations.GetEmoteRelationshipOwner(aggregations.UserRelationshipOptions{Roles: true, Editors: true})...))
		err = multierror.Append(err, cur.All(ctx, &targetEmotes)).ErrorOrNil()
		if err != nil {
			return errors.ErrUnknownEmote()
		}
		for _, e := range targetEmotes {
			for _, ver := range e.Versions {
				if v, ok := targetEmoteMap[ver.ID]; ok {
					v.emote = e
					targetEmoteMap[e.ID] = v
				}
			}
		}

		// Fetch set owner
		owner, err := m.loaders.UserByID().Load(set.OwnerID)
		if err == nil {
			set.Owner = &owner
		}
	}

	// The actor must have access to the emote set
	if set.OwnerID != actor.ID && !actor.HasPermission(structures.RolePermissionEditAnyEmoteSet) {
		if set.Privileged && !actor.HasPermission(structures.RolePermissionSuperAdministrator) {
			return errors.ErrInsufficientPrivilege().SetDetail("This set is privileged")
		}

		if set.Owner != nil {
			ed, ok, _ := set.Owner.GetEditor(actor.ID)
			if !ok {
				return errors.ErrInsufficientPrivilege().SetDetail("You do not have permission to modify this emote set")
			}

			if !ed.HasPermission(structures.UserEditorPermissionModifyEmotes) {
				return errors.ErrInsufficientPrivilege().SetDetail("You do not have permission to change content in this emote set").SetFields(errors.Fields{
					"MISSING_EDITOR_PERMISSION": "MODIFY_EMOTES",
				})
			}
		}
	}

	// Make a map of active set emotes
	activeEmotes := map[primitive.ObjectID]*structures.Emote{}
	for _, e := range set.Emotes {
		activeEmotes[e.ID] = e.Emote
	}

	// Set up audit log entry
	c := &structures.AuditLogChange{
		Format: structures.AuditLogChangeFormatArrayChange,
		Key:    "emotes",
	}
	log := structures.NewAuditLogBuilder(structures.AuditLog{}).
		SetKind(structures.AuditLogKindUpdateEmoteSet).
		SetActor(actor.ID).
		SetTargetKind(structures.ObjectKindEmoteSet).
		SetTargetID(set.ID).
		AddChanges(c)

	// get end pos
	endPos := len(set.Emotes)

	for i, ae := range set.Emotes {
		if !ae.Origin.ID.IsZero() {
			endPos = i
			break
		}
	}

	emotes := make([]structures.ActiveEmote, endPos)
	copy(emotes, set.Emotes[:endPos])

	// Iterate through the target emotes
	// Check for permissions
	for _, tgt := range targetEmoteMap {
		if tgt.emote == nil {
			continue
		}

		z := zap.S().With(
			"EMOTE_ID", tgt.ID.Hex(),
			"ACTOR_ID", actor.ID.Hex(),
		)

		initName := tgt.emote.Name

		tgt.Name = utils.Ternary(tgt.Name != "", tgt.Name, tgt.emote.Name)
		tgt.emote.Name = tgt.Name

		// Reject bad name if the action isn't REMOVE
		if err := tgt.emote.Validator().Name(); err != nil && tgt.Action != structures.ListItemActionRemove {
			return err
		}

		tgt.emote.Name = initName

		switch tgt.Action {
		// ADD EMOTE
		case structures.ListItemActionAdd:
			// Handle emote privacy
			if tgt.emote.Flags.Has(structures.EmoteFlagsPrivate) {
				usable := false
				// Usable if actor has Bypass Privacy permission
				if actor.HasPermission(structures.RolePermissionBypassPrivacy) {
					usable = true
				}
				// Usable if actor is an editor of emote owner
				// and has the correct permission
				if tgt.emote.Owner != nil {
					var editor structures.UserEditor

					for _, ed := range tgt.emote.Owner.Editors {
						if opt.Actor.ID == ed.ID {
							editor = ed
							break
						}
					}

					if !editor.ID.IsZero() && editor.HasPermission(structures.UserEditorPermissionUsePrivateEmotes) {
						usable = true
					}
				}

				if !usable {
					return errors.ErrInsufficientPrivilege().SetFields(errors.Fields{
						"EMOTE_ID": tgt.ID.Hex(),
					}).SetDetail("Private Emote")
				}
			}

			// Personal Set: if emote is not personal use-approved, send a new mod request
			if set.Flags.Has(structures.EmoteSetFlagPersonal) && !tgt.emote.Flags.Has(structures.EmoteFlagsPersonalUse) {
				if tgt.emote.Flags.Has(structures.EmoteFlagsPersonalUseRejected) {
					return errors.ErrInsufficientPrivilege().SetFields(errors.Fields{
						"EMOTE_ID": tgt.ID.Hex(),
					}).SetDetail("This emote has been rejected for use in personal sets")
				}

				// Acquire a lock. This prevents multiple personal use requests from being sent for the same emote
				mx := m.redis.Mutex(m.redis.ComposeKey("api", "emote", tgt.ID.Hex(), "mod_request", "personal_use"), time.Minute)
				if err := mx.LockContext(ctx); err != nil {
					z.Errorw("failed to acquire lock", "error", err)
				}

				// Check for existing mod request
				cur, err := m.mongo.Collection(mongo.CollectionNameMessages).Aggregate(ctx, mongo.Pipeline{
					{{
						Key: "$match",
						Value: bson.M{ // find mod request messages targeting this emote
							"kind":             structures.MessageKindModRequest,
							"data.target_kind": structures.ObjectKindEmote,
							"data.target_id":   tgt.emote.ID,
						},
					}},
					{{
						Key: "$lookup",
						Value: mongo.Lookup{ // join with read states
							From:         mongo.CollectionNameMessagesRead,
							LocalField:   "_id",
							ForeignField: "message_id",
							As:           "read_states",
						},
					}},
					{{
						Key: "$project", // return true if read states exist
						Value: bson.M{"is_pending": bson.M{
							"$gt": bson.A{bson.M{"$size": bson.M{"$filter": bson.M{
								"input": "$read_states",
								"as":    "rs",
								"cond":  bson.M{"$eq": bson.A{"$$rs.read", false}},
							}}}, 0},
						}},
					}},
				})
				if err != nil {
					z.Errorw("failed to check for existing mod request (aggregation couldn't be formulated)")

					_, _ = mx.UnlockContext(ctx)

					return errors.ErrInternalServerError()
				}

				states := []struct {
					IsPending bool `bson:"is_pending"`
				}{}
				if err := cur.All(ctx, &states); err != nil {
					z.Errorw("failed to check for existing mod request (could not return from cursor)")

					_, _ = mx.UnlockContext(ctx)

					return errors.ErrInternalServerError()
				}

				// If there is a pending mod request, skip
				alreadyRequested := false

				for _, state := range states {
					if alreadyRequested = state.IsPending; alreadyRequested {
						break
					}
				}

				if !alreadyRequested {
					// Construct & write new mod request
					mb := structures.NewMessageBuilder(structures.Message[structures.MessageDataModRequest]{}).
						SetKind(structures.MessageKindModRequest).
						SetAuthorID(actor.ID).
						SetData(structures.MessageDataModRequest{
							TargetKind: structures.ObjectKindEmote,
							TargetID:   tgt.emote.ID,
							Wish:       "personal_use",
						})
					if err := m.SendModRequestMessage(ctx, mb); err != nil {
						z.Errorw("failed to send personal_use mod request message", "error", err)
					}
				}

				_, _ = mx.UnlockContext(ctx)
			}

			// Check zero-width permission
			if set.Owner == nil || tgt.emote.Flags.Value()&structures.EmoteFlagsZeroWidth != 0 && !set.Owner.HasPermission(structures.RolePermissionFeatureZeroWidthEmoteType) {
				return errors.ErrInsufficientPrivilege().SetDetail("You must be a subscriber to use zero-width emotes")
			}

			// Verify that the set has available slots
			if !actor.HasPermission(structures.RolePermissionEditAnyEmoteSet) {
				if len(emotes) >= int(set.Capacity) {
					return errors.ErrNoSpaceAvailable().
						SetDetail("This set does not have enough slots").
						SetFields(errors.Fields{"CAPACITY": set.Capacity})
				}
			}

			// Check for conflicts with existing emotes
			for _, e := range emotes {
				// Cannot enable the same emote twice
				if tgt.ID == e.ID {
					return errors.ErrEmoteAlreadyEnabled()
				}
				// Cannot have the same emote name as another active emote
				if tgt.Name == e.Name {
					return errors.ErrEmoteNameConflict()
				}
			}

			// Add active emote
			at := time.Now()
			esb.AddActiveEmote(tgt.ID, tgt.Name, at, &actor.ID)
			c.WriteArrayAdded(structures.ActiveEmote{
				ID:        tgt.ID,
				Name:      tgt.Name,
				Flags:     tgt.Flags,
				Timestamp: at,
				ActorID:   actor.ID,
			})

			// Publish a message to the Event API
			_ = m.events.Publish(ctx, events.NewMessage(events.OpcodeDispatch, events.DispatchPayload{
				Type: events.EventTypeUpdateEmoteSet,
				Conditions: []events.EventCondition{{
					"object_id": esb.EmoteSet.ID.Hex(),
				}},
				Body: events.ChangeMap{
					ID:    esb.EmoteSet.ID,
					Kind:  structures.ObjectKindEmoteSet,
					Actor: m.modelizer.User(actor),
					Pushed: []events.ChangeField{{
						Key:   "emotes",
						Index: utils.PointerOf(int32(endPos)),
						Type:  events.ChangeFieldTypeObject,
						Value: m.modelizer.ActiveEmote(structures.ActiveEmote{
							ID:        tgt.ID,
							Name:      tgt.Name,
							Flags:     tgt.Flags,
							Timestamp: at,
							ActorID:   actor.ID,
							Emote:     tgt.emote,
						}),
					}},
				},
			}).ToRaw())
		case structures.ListItemActionUpdate, structures.ListItemActionRemove:
			// The emote must already be active
			found := false

			var (
				ind int
				ae  structures.ActiveEmote
			)

			for ind, ae = range emotes {
				if tgt.Action == structures.ListItemActionUpdate && ae.Name == tgt.Name {
					return errors.ErrEmoteNameConflict().SetFields(errors.Fields{
						"EMOTE_ID":          tgt.ID.Hex(),
						"CONFLICT_EMOTE_ID": tgt.ID.Hex(),
					})
				}

				if ae.ID == tgt.ID {
					found = true
					break
				}
			}

			if !found {
				return errors.ErrEmoteNotEnabled().SetFields(errors.Fields{
					"EMOTE_ID": tgt.ID.Hex(),
				})
			}

			// In Personal Sets: the emote cannot be modified or deleted before the timer
			if set.Flags.Has(structures.EmoteSetFlagPersonal) {
				at := ae.Timestamp
				if ae.TimestampUpdate != nil && ae.TimestampUpdate.After(at) {
					at = *ae.TimestampUpdate
				}

				thresold := at.Add(EMOTE_SET_PERSONAL_MIN_EMOTE_LIFETIME)

				if thresold.After(time.Now()) {
					return errors.ErrRateLimited().SetDetail(
						"This emote is locked in this set for %s minutes",
						strconv.FormatFloat(-time.Since(thresold).Minutes(), 'f', 0, 64),
					)
				}
			}

			if tgt.Action == structures.ListItemActionUpdate {
				// Modify active emote
				c.WriteArrayUpdated(structures.AuditLogChangeSingleValue{
					New: structures.ActiveEmote{
						ID:              tgt.ID,
						Name:            tgt.Name,
						Flags:           tgt.Flags,
						Timestamp:       ae.Timestamp,
						TimestampUpdate: utils.PointerOf(time.Now()),
					},
					Old: structures.ActiveEmote{
						ID:              ae.ID,
						Name:            ae.Name,
						Flags:           ae.Flags,
						Timestamp:       ae.Timestamp,
						TimestampUpdate: ae.TimestampUpdate,
					},
					Position: int32(ind),
				})
				esb.UpdateActiveEmote(tgt.ID, tgt.Name)

				_ = m.events.Publish(ctx, events.NewMessage(events.OpcodeDispatch, events.DispatchPayload{
					Type: events.EventTypeUpdateEmoteSet,
					Conditions: []events.EventCondition{{
						"object_id": esb.EmoteSet.ID.Hex(),
					}},
					Body: events.ChangeMap{
						ID:    esb.EmoteSet.ID,
						Kind:  structures.ObjectKindEmoteSet,
						Actor: m.modelizer.User(actor),
						Updated: []events.ChangeField{{
							Key:   "emotes",
							Index: utils.PointerOf(int32(ind)),
							Type:  events.ChangeFieldTypeObject,
							OldValue: m.modelizer.ActiveEmote(structures.ActiveEmote{
								ID:        ae.ID,
								Name:      ae.Name,
								Flags:     ae.Flags,
								Timestamp: ae.Timestamp,
								ActorID:   ae.ActorID,
							}),
							Value: m.modelizer.ActiveEmote(structures.ActiveEmote{
								ID:        tgt.ID,
								Name:      tgt.Name,
								Flags:     tgt.Flags,
								Timestamp: ae.Timestamp,
								ActorID:   actor.ID,
								Emote:     tgt.emote,
							}),
						}},
					},
				}).ToRaw())
			} else if tgt.Action == structures.ListItemActionRemove {
				// Remove active emote
				_, ind := esb.RemoveActiveEmote(tgt.ID)
				c.WriteArrayRemoved(structures.ActiveEmote{
					ID: tgt.ID,
				})

				_ = m.events.Publish(ctx, events.NewMessage(events.OpcodeDispatch, events.DispatchPayload{
					Type: events.EventTypeUpdateEmoteSet,
					Conditions: []events.EventCondition{{
						"object_id": esb.EmoteSet.ID.Hex(),
					}},
					Body: events.ChangeMap{
						ID:    esb.EmoteSet.ID,
						Kind:  structures.ObjectKindEmoteSet,
						Actor: m.modelizer.User(actor),
						Pulled: []events.ChangeField{{
							Key:   "emotes",
							Index: utils.PointerOf(int32(ind)),
							Type:  events.ChangeFieldTypeObject,
							OldValue: m.modelizer.ActiveEmote(structures.ActiveEmote{
								ID:      tgt.ID,
								Name:    tgt.Name,
								ActorID: actor.ID,
							}),
						}},
					},
				}).ToRaw())
			}
		}
	}

	// Update the document
	if len(esb.Update) == 0 {
		return errors.ErrUnknownEmote().SetDetail("no target emotes found")
	}

	if err := m.mongo.Collection(mongo.CollectionNameEmoteSets).FindOneAndUpdate(
		ctx,
		bson.M{"_id": set.ID},
		esb.Update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&esb.EmoteSet); err != nil {
		zap.S().Errorw("mongo, failed to write changes to emote set",
			"emote_set_id", esb.EmoteSet.ID.Hex(),
		)

		return errors.ErrInternalServerError()
	}

	// Write audit log entry
	if _, err := m.mongo.Collection(mongo.CollectionNameAuditLogs).InsertOne(ctx, log.AuditLog); err != nil {
		zap.S().Errorw("mongo, failed to write audit log entry for changes to emote set",
			"emote_set_id", esb.EmoteSet.ID.Hex(),
		)
	}

	esb.MarkAsTainted()

	return nil
}
