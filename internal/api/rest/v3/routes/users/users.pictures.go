package users

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awss3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/h2non/filetype/matchers"
	"github.com/seventv/api/internal/api/rest/middleware"
	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/svc/s3"
	"github.com/seventv/common/utils"
	"github.com/seventv/image-processor/go/container"
	"github.com/seventv/image-processor/go/task"
	messagequeue "github.com/seventv/message-queue/go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type pictureUploadRoute struct {
	Ctx global.Context
}

func newPictureUpload(gctx global.Context) rest.Route {
	userPictureListener(gctx)
	return &pictureUploadRoute{gctx}
}

func (r *pictureUploadRoute) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:      "/{user.id}/profile-picture",
		Method:   rest.PUT,
		Children: []rest.Route{},
		Middleware: []rest.Middleware{
			middleware.Auth(r.Ctx, true),
			middleware.RateLimit(r.Ctx, "UpdateUserPicture", [2]int64{2, 60}),
		},
	}
}

type pictureTaskMetadata struct {
	UserID          primitive.ObjectID `json:"user_id"`
	InputFileKey    string             `json:"input_file_key"`
	InputFileBucket string             `json:"input_file_bucket"`
}

// @Summary Upload Profile Picture
// @Description Set a new profile picture
// @Tags users
// @Accept image/avif,image/webp,image/gif,image/apng,image/png,image/jpeg
// @Success 200
// @Router /users/{user.id}/profile-picture [put]
func (r *pictureUploadRoute) Handler(ctx *rest.Ctx) rest.APIError {
	done := r.Ctx.Inst().Limiter.AwaitMutation(ctx)
	defer done()

	actor, ok := ctx.GetActor()
	if !ok {
		return errors.ErrUnauthorized()
	}

	victimID, _ := ctx.UserValue("user.id").String()

	ctx.SetContentType("application/json")

	var victim structures.User

	switch victimID {
	case "@me":
		victim = actor
	default:
		oid, err := ctx.UserValue("user").ObjectID()
		if err != nil {
			return errors.From(err)
		}

		victim, err = r.Ctx.Inst().Loaders.UserByID().Load(oid)
		if err != nil {
			return errors.From(err)
		}
	}

	// Permission check
	if actor.ID != victim.ID && !actor.HasPermission(structures.RolePermissionManageUsers) {
		noPrivilege := errors.ErrInsufficientPrivilege().SetDetail("You are not allowed to perform this action on this user")

		ed, ok, _ := victim.GetEditor(actor.ID)
		if !ok {
			return noPrivilege
		}

		if !ed.HasPermission(structures.UserEditorPermissionManageProfile) {
			return noPrivilege.SetFields(errors.Fields{
				"MISSING_EDITOR_PERMISSION": "MANAGE_PROFILE",
			})
		}
	}

	req := &ctx.Request

	body := req.Body()

	fileType := container.Match(body)
	switch fileType {
	case container.TypeAvif:
	case matchers.TypeWebp:
	case matchers.TypeGif:
	case matchers.TypePng:
	case matchers.TypeJpeg:
	default:
		return errors.ErrInvalidRequest().SetDetail(fmt.Sprintf("Bad emote upload type '%s'", fileType.MIME.Value))
	}

	id := primitive.NewObjectIDFromTimestamp(time.Now())

	rawFilekey := r.Ctx.Inst().S3.ComposeKey("user", victim.ID.Hex(), fmt.Sprintf("av_%s", id.Hex()), fmt.Sprintf("input.%s", fileType.Extension))

	if err := r.Ctx.Inst().S3.UploadFile(
		ctx,
		&awss3.PutObjectInput{
			Body:         aws.ReadSeekCloser(bytes.NewReader(body)),
			Key:          aws.String(rawFilekey),
			ACL:          s3.AclPrivate,
			Bucket:       aws.String(r.Ctx.Config().S3.InternalBucket),
			ContentType:  aws.String(fileType.MIME.Value),
			CacheControl: s3.DefaultCacheControl,
		},
	); err != nil {
		zap.S().Errorw("failed to upload image to s3",
			"error", err,
		)

		return errors.ErrMissingInternalDependency().SetDetail("Failed to establish connection with the CDN Service")
	}

	allowAnim := actor.HasPermission(structures.RolePermissionFeatureProfilePictureAnimation)

	victimIDBytes, _ := json.Marshal(pictureTaskMetadata{
		UserID:          victim.ID,
		InputFileKey:    rawFilekey,
		InputFileBucket: r.Ctx.Config().S3.InternalBucket,
	})

	taskData, err := json.Marshal(task.Task{
		ID:    id.Hex(),
		Flags: utils.Ternary(allowAnim, task.TaskFlagWEBP|task.TaskFlagAVIF|task.TaskFlagGIF, 0) | task.TaskFlagWEBP_STATIC | task.TaskFlagAVIF_STATIC | task.TaskFlagPNG | task.TaskFlagPNG_STATIC,
		Input: task.TaskInput{
			Bucket: r.Ctx.Config().S3.InternalBucket,
			Key:    rawFilekey,
		},
		Output: task.TaskOutput{
			Prefix:       r.Ctx.Inst().S3.ComposeKey("user", victim.ID.Hex(), fmt.Sprintf("av_%s", id.Hex())),
			Bucket:       r.Ctx.Config().S3.PublicBucket,
			CacheControl: *s3.DefaultCacheControl,
		},
		SmallestMaxWidth:  48,
		SmallestMaxHeight: 48,
		Scales:            []int{1, 2, 3},
		ResizeRatio:       task.ResizeRatioPaddingCenter,
		Limits: task.TaskLimits{
			MaxProcessingTime: time.Duration(r.Ctx.Config().Limits.Emotes.MaxProcessingTimeSeconds) * time.Second,
			MaxFrameCount:     500,
			MaxWidth:          r.Ctx.Config().Limits.Emotes.MaxWidth,
			MaxHeight:         r.Ctx.Config().Limits.Emotes.MaxHeight,
		},
		Metadata: victimIDBytes,
	})
	if err == nil {
		err = r.Ctx.Inst().MessageQueue.Publish(ctx, messagequeue.OutgoingMessage{
			Queue:   r.Ctx.Config().MessageQueue.ImageProcessorJobsQueueName,
			Headers: map[string]string{},
			Flags: messagequeue.MessageFlags{
				ID:          id.Hex(),
				ContentType: "application/json",
				ReplyTo:     r.Ctx.Config().MessageQueue.ImageProcessorUserPicturesResultsQueueName,
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
		zap.S().Errorw("failed to set up image processing task for user picture", "error", err)

		return errors.ErrInternalServerError().SetDetail("Task creation failed")
	}

	// Set pending profile picture
	if _, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).UpdateOne(ctx, bson.M{
		"_id": victim.ID,
	}, bson.M{
		"$set": bson.M{"avatar.pending_id": id},
	}); err != nil {
		zap.S().Errorw("mongo, failed to update user state with pending profile picture", "error", err)
		return errors.ErrInternalServerError()
	}

	return ctx.JSON(rest.OK, struct{}{})
}
