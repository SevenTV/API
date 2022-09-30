package mutate

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func (m *Mutate) DeleteEmote(ctx context.Context, eb *structures.EmoteBuilder, opt DeleteEmoteOptions) error {
	if eb == nil {
		return errors.ErrInternalIncompleteMutation()
	} else if eb.IsTainted() {
		return errors.ErrMutateTaintedObject()
	}

	if opt.Undo {
		return fmt.Errorf("TODO error unimplemented emote deletion undo")
	}

	// Check permissions
	actor := opt.Actor
	// Check actor's permission
	if actor != nil {
		// User is not privileged
		if !actor.HasPermission(structures.RolePermissionEditAnyEmote) {
			if eb.Emote.OwnerID.IsZero() { // Deny when emote has no owner
				return errors.ErrInsufficientPrivilege()
			}

			// Check if actor is editor of the emote owner
			isPermittedEditor := false

			for _, ed := range actor.EditorOf {
				if ed.ID != eb.Emote.OwnerID {
					continue
				}
				// Allow if the actor has the "manage owned emotes" permission
				// as the editor of the emote owner
				if ed.HasPermission(structures.UserEditorPermissionManageOwnedEmotes) {
					isPermittedEditor = true
					break
				}
			}

			if eb.Emote.OwnerID != actor.ID && !isPermittedEditor { // Deny when not the owner or editor of the owner of the emote
				return errors.ErrInsufficientPrivilege()
			}
		}

		if opt.Undo && !actor.HasPermission(structures.RolePermissionEditAnyEmote) {
			return errors.ErrInsufficientPrivilege().SetDetail("You cannot reinstate an emote after it has been deleted")
		}
	} else if !opt.SkipValidation {
		// if validation is not skipped then an Actor is mandatory
		return errors.ErrUnauthorized()
	}

	setACL := func(v structures.EmoteVersion, acl string) structures.EmoteVersion {
		wg := sync.WaitGroup{}
		wg.Add(len(v.ImageFiles) + 1)

		for _, f := range append(v.ImageFiles, v.ArchiveFile) {
			// Set object ACL
			go func(f structures.ImageFile) {
				if err := m.s3.DeleteFile(ctx, &s3.DeleteObjectInput{
					Bucket: &f.Bucket,
					Key:    &f.Key,
				}); err != nil {
					zap.S().Errorw("s3, failed to set delete file emote during its deletion",
						"error", err,
						"emote_id", eb.Emote.ID,
						"version_id", v.ID,
						"s3_bucket", f.Bucket,
						"s3_object_key", f.Key,
					)
				}

				wg.Done()
			}(f)
		}

		wg.Wait()

		return v
	}

	versionIDs := make([]primitive.ObjectID, len(eb.Emote.Versions))

	// Mark the emote as deleted
	acl := utils.Ternary(opt.Undo, "public-read", "private")
	lifecycle := utils.Ternary(opt.Undo, structures.EmoteLifecycleLive, structures.EmoteLifecycleDeleted)

	if opt.VersionID.IsZero() {
		for i, ver := range eb.Emote.Versions {
			ver.State.Lifecycle = lifecycle
			ver = setACL(ver, acl)
			eb.UpdateVersion(ver.ID, ver)

			versionIDs[i] = ver.ID
		}
	} else {
		ver, _ := eb.Emote.GetVersion(opt.VersionID)
		if ver.ID.IsZero() {
			return errors.ErrUnknownEmote().SetDetail("Specified version does not exist")
		}
		ver.State.Lifecycle = lifecycle
		ver = setACL(ver, acl)
		eb.UpdateVersion(ver.ID, ver)
	}

	// Write the update to the emote lifecycle
	if _, err := m.mongo.Collection(mongo.CollectionNameEmotes).UpdateOne(ctx, bson.M{
		"versions.id": eb.Emote.ID,
	}, eb.Update); err != nil {
		zap.S().Errorw("mongo, failed to update emote during its deletion",
			"error", err,
		)

		return errors.ErrInternalServerError()
	}

	// Remove any mod requests for the emote
	// if !opt.Undo {
	// 	_, err := m.mongo.Collection(mongo.CollectionNameMessages).DeleteMany(ctx, bson.M{
	// 		"kind":             structures.MessageKindModRequest,
	// 		"data.target_kind": structures.ObjectKindEmote,
	// 		"data.target_id":   utils.Ternary(opt.VersionID.IsZero(), bson.M{"$in": versionIDs}, bson.M{"$eq": opt.VersionID}),
	// 	})
	// 	if err != nil {
	// 		zap.S().Errorw("mongo, failed to delete mod requests for emote during its deletion", "error", err)
	// 	}
	// }

	// Write audit log
	alb := structures.NewAuditLogBuilder(structures.AuditLog{
		Changes: []*structures.AuditLogChange{},
		Reason:  opt.Reason,
	}).
		SetKind(utils.Ternary(opt.Undo, structures.AuditLogKindUndoDeleteEmote, structures.AuditLogKindDeleteEmote)).
		SetActor(actor.ID).
		SetTargetKind(structures.ObjectKindEmote).
		SetTargetID(eb.Emote.ID)

	if _, err := m.mongo.Collection(mongo.CollectionNameAuditLogs).InsertOne(ctx, alb.AuditLog); err != nil {
		zap.S().Errorw("mongo, failed to write audit log during emote deletion",
			"emote_id", eb.Emote.ID,
			"error", err,
		)
	}

	return nil
}

type DeleteEmoteOptions struct {
	Actor *structures.User
	// If true, this is a un-delete operation
	Undo bool
	// If specified, only this version will be deleted
	//
	// by default, all versions will be deleted
	VersionID primitive.ObjectID
	// The reason given for the deletion: will appear in audit logs
	Reason         string
	SkipValidation bool
}
