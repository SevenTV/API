package emote

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"time"

	"github.com/seventv/api/data/mutate"
	"github.com/seventv/api/internal/events"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/api/internal/gql/v3/types"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/svc/s3"
	"github.com/seventv/common/utils"
	"github.com/seventv/image-processor/go/task"
	messagequeue "github.com/seventv/message-queue/go"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

type ResolverOps struct {
	types.Resolver
}

func NewOps(r types.Resolver) generated.EmoteOpsResolver {
	return &ResolverOps{r}
}

func (r *ResolverOps) Rerun(ctx context.Context, obj *model.EmoteOps) (*model.Emote, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() || !actor.HasPermission(structures.RolePermissionRunJobs) {
		return nil, errors.ErrInsufficientPrivilege()
	}

	// Get the emote
	emote, err := r.Ctx.Inst().Loaders.EmoteByID().Load(obj.ID)
	if err != nil {
		return nil, err
	}

	ver, verIndex := emote.GetVersion(emote.ID)
	if verIndex == -1 {
		return nil, errors.ErrUnknownEmote()
	}
	// Update the emote's lifecycle
	eb := structures.NewEmoteBuilder(emote)

	// Send the task
	filekey := ver.InputFile.Key
	taskData, err := json.Marshal(task.Task{
		ID:    emote.ID.Hex(),
		Flags: task.TaskFlagALL,
		Input: task.TaskInput{
			Bucket: r.Ctx.Config().S3.InternalBucket,
			Key:    filekey,
		},
		Output: task.TaskOutput{
			Prefix:       r.Ctx.Inst().S3.ComposeKey("emote", path.Dir(ver.InputFile.Key)),
			Bucket:       r.Ctx.Config().S3.PublicBucket,
			ACL:          *s3.AclPublicRead,
			CacheControl: *s3.DefaultCacheControl,
		},
		SmallestMaxWidth:  96,
		SmallestMaxHeight: 32,
		Scales:            []int{1, 2, 3, 4},
		ResizeRatio:       task.ResizeRatioNothing,
		Limits: task.TaskLimits{
			MaxProcessingTime: time.Duration(r.Ctx.Config().Limits.Emotes.MaxProcessingTimeSeconds) * time.Second,
			MaxFrameCount:     r.Ctx.Config().Limits.Emotes.MaxFrameCount,
			MaxWidth:          r.Ctx.Config().Limits.Emotes.MaxWidth,
			MaxHeight:         r.Ctx.Config().Limits.Emotes.MaxHeight,
		},
	})

	if err == nil {
		err = r.Ctx.Inst().MessageQueue.Publish(ctx, messagequeue.OutgoingMessage{
			Queue:   r.Ctx.Config().MessageQueue.ImageProcessorJobsQueueName,
			Headers: messagequeue.MessageHeaders{},
			Flags: messagequeue.MessageFlags{
				ID:          emote.ID.Hex(),
				ContentType: "application/json",
				ReplyTo:     r.Ctx.Config().MessageQueue.ImageProcessorResultsQueueName,
				Timestamp:   time.Now(),
				RMQ: messagequeue.MessageFlagsRMQ{
					DeliveryMode: messagequeue.RMQDeliveryModePersistent,
				},
				SQS: messagequeue.MessageFlagsSQS{},
			},
			Body: taskData,
		})
	}

	if err != nil {
		zap.S().Errorw("failed to marshal task",
			"error", err,
		)

		return nil, errors.ErrInternalServerError()
	}

	eb.Update.Set(fmt.Sprintf("versions.%d.state.lifecycle", verIndex), structures.EmoteLifecycleProcessing)

	if _, err = r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmotes).UpdateOne(ctx, bson.M{
		"versions.id": emote.ID,
	}, eb.Update); err != nil {
		zap.S().Errorw("mongo, failed to update lifecycle of emote pending processing job rerun")
	}

	if err = r.Ctx.Inst().Redis.RawClient().Publish(ctx, fmt.Sprintf("events:sub:emotes:%s", emote.ID.Hex()), "1").Err(); err != nil {
		return nil, errors.ErrInternalServerError()
	}

	return helpers.EmoteStructureToModel(emote, r.Ctx.Config().CdnURL), nil
}

func (r *ResolverOps) Update(ctx context.Context, obj *model.EmoteOps, params model.EmoteUpdate, reason *string) (*model.Emote, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	emotes, err := r.Ctx.Inst().Query.Emotes(ctx, bson.M{"versions.id": obj.ID}).Items()
	if err != nil {
		return nil, err
	}

	if len(emotes) == 0 {
		return nil, errors.ErrUnknownEmote()
	}

	emote := emotes[0]
	ver, _ := emote.GetVersion(obj.ID)
	eb := structures.NewEmoteBuilder(emote)

	// Cannot edit deleted version without privileges
	if !actor.HasPermission(structures.RolePermissionEditAnyEmote) && ver.IsUnavailable() {
		return nil, errors.ErrUnknownEmote()
	}

	if ver.IsProcessing() {
		return nil, errors.ErrInsufficientPrivilege().SetDetail("Cannot edit emote in a processing state")
	}

	// Edit listed (version)
	versionUpdated := false

	// Reason
	rsn := ""
	if reason != nil {
		rsn = *reason
	}

	// Delete emote
	// no other params can be used if `deleted` is true
	if params.Deleted != nil {
		del := *params.Deleted

		err = r.Ctx.Inst().Mutate.DeleteEmote(ctx, eb, mutate.DeleteEmoteOptions{
			Actor:     &actor,
			VersionID: obj.ID,
			Undo:      !del,
			Reason:    rsn,
		})

		if err != nil {
			return nil, err
		}
	} else {
		// Edit name
		if params.Name != nil {
			eb.SetName(*params.Name)
		}
		// Edit owner
		if params.OwnerID != nil {
			eb.SetOwnerID(*params.OwnerID)
		}
		// Edit tags
		if params.Tags != nil {
			if !actor.HasPermission(structures.RolePermissionManageContent) {
				for _, tag := range params.Tags {
					if utils.Contains(r.Ctx.Config().Limits.Emotes.ReservedTags, tag) {
						return nil, errors.ErrInsufficientPrivilege().SetDetail("You cannot use reserved tag #%s", tag)
					}
				}
			}

			eb.SetTags(params.Tags, true)
		}
		// Edit flags
		if params.Flags != nil {
			f := structures.EmoteFlag(*params.Flags)
			eb.SetFlags(f)
		}

		if params.Listed != nil {
			ver.State.Listed = *params.Listed
			versionUpdated = true
		}

		if params.VersionName != nil {
			ver.Name = *params.VersionName
			versionUpdated = true
		}

		if params.VersionDescription != nil {
			ver.Description = *params.VersionDescription
			versionUpdated = true
		}

		if versionUpdated {
			eb.UpdateVersion(obj.ID, ver)
		}

		if err := r.Ctx.Inst().Mutate.EditEmote(ctx, eb, mutate.EmoteEditOptions{
			Actor: actor,
		}); err != nil {
			return nil, err
		}
	}

	go func() {
		events.Publish(r.Ctx, "emotes", obj.ID)
	}()

	emote, err = r.Ctx.Inst().Loaders.EmoteByID().Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return helpers.EmoteStructureToModel(emote, r.Ctx.Config().CdnURL), nil
}
