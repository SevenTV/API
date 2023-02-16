package emote

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"time"

	"github.com/seventv/api/data/model/modelgql"
	"github.com/seventv/api/internal/api/gql/v3/auth"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/svc/s3"
	"github.com/seventv/image-processor/go/task"
	messagequeue "github.com/seventv/message-queue/go"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

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

	return modelgql.EmoteModel(r.Ctx.Inst().Modelizer.Emote(emote)), nil
}
