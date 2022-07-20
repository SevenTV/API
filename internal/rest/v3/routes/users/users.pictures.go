package users

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/google/uuid"
	"github.com/h2non/filetype/matchers"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/middleware"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/svc/s3"
	"github.com/seventv/image-processor/go/container"
	"github.com/seventv/image-processor/go/task"
	messagequeue "github.com/seventv/message-queue/go"
	"go.mongodb.org/mongo-driver/bson"
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
		URI:      "/{user}/profile-picture",
		Method:   rest.PUT,
		Children: []rest.Route{},
		Middleware: []rest.Middleware{
			middleware.Auth(r.Ctx),
		},
	}
}

// Submit Profile Picture
// @Summary Submit Profile Picture
// @Description Set a new profile picture
// @Tags users
// @Accept image/avif, image/webp, image/gif, image/apng, image/png, image/jpeg
// @Success 200 {object} model.User
// @Router /users/{user}/profile-picture [put]
func (r *pictureUploadRoute) Handler(ctx *rest.Ctx) rest.APIError {
	done := r.Ctx.Inst().Limiter.AwaitMutation(ctx)
	defer done()

	actor, ok := ctx.GetActor()
	if !ok {
		return errors.ErrUnauthorized()
	}

	victimID, _ := ctx.UserValue("user").String()

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

	id := uuid.New()
	idb, _ := id.MarshalBinary()
	strId := hex.EncodeToString(idb)

	rawFilekey := r.Ctx.Inst().S3.ComposeKey("pp", victim.ID.Hex(), fmt.Sprintf("%s_raw.%s", strId, fileType.Extension))

	if err := r.Ctx.Inst().S3.UploadFile(
		ctx,
		&s3manager.UploadInput{
			Body:         bytes.NewBuffer(body),
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

	taskData, err := json.Marshal(task.Task{
		ID:    strId,
		Flags: task.TaskFlagWEBP | task.TaskFlagWEBP_STATIC,
		Input: task.TaskInput{
			Bucket: r.Ctx.Config().S3.InternalBucket,
			Key:    rawFilekey,
		},
		Output: task.TaskOutput{
			Prefix:               r.Ctx.Inst().S3.ComposeKey("pp", victim.ID.Hex()),
			Bucket:               r.Ctx.Config().S3.PublicBucket,
			ACL:                  *s3.AclPrivate,
			CacheControl:         *s3.DefaultCacheControl,
			ExcludeFileExtension: true,
		},
		SmallestMaxWidth:  128,
		SmallestMaxHeight: 128,
		Scales:            []int{1},
		ResizeRatio:       task.ResizeRatioPaddingCenter,
		Limits: task.TaskLimits{
			MaxProcessingTime: time.Duration(r.Ctx.Config().Limits.Emotes.MaxProcessingTimeSeconds) * time.Second,
			MaxFrameCount:     500,
			MaxWidth:          r.Ctx.Config().Limits.Emotes.MaxWidth,
			MaxHeight:         r.Ctx.Config().Limits.Emotes.MaxHeight,
		},
	})
	if err == nil {
		err = r.Ctx.Inst().MessageQueue.Publish(ctx, messagequeue.OutgoingMessage{
			Queue:   r.Ctx.Config().MessageQueue.ImageProcessorJobsQueueName,
			Headers: map[string]string{},
			Flags: messagequeue.MessageFlags{
				ID:          strId,
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
		"$set": bson.M{"state.pending_avatar_id": strId},
	}); err != nil {
		zap.S().Errorw("mongo, failed to update user state with pending profile picture", "error", err)
		return errors.ErrInternalServerError()
	}

	return ctx.JSON(rest.OK, nil)
}
