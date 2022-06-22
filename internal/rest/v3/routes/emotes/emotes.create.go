package emotes

import (
	"bytes"
	"fmt"
	"path"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	jsoniter "github.com/json-iterator/go"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/api/internal/rest/v3/middleware"
	"github.com/seventv/api/internal/rest/v3/model"
	"github.com/seventv/api/internal/svc/s3"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"github.com/seventv/image-processor/go/container"
	"github.com/seventv/image-processor/go/task"
	messagequeue "github.com/seventv/message-queue/go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type create struct {
	Ctx global.Context
}

func newCreate(gCtx global.Context) rest.Route {
	return &create{gCtx}
}

func (r *create) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "",
		Method: rest.POST,
		Middleware: []rest.Middleware{
			middleware.Auth(r.Ctx),
		},
	}
}

// Create Emote
// @Summary Create Emote
// @Description Upload a new emote
// @Tags emotes
// @Accept image/webp, image/gif, image/png, image/apng, image/avif, image/jpeg, image/tiff, image/webm
// @Param X-Emote-Data header string false "Initial emote properties"
// @Produce json
// @Success 201 {object} model.Emote
// @Router /emotes [post]
func (r *create) Handler(ctx *rest.Ctx) rest.APIError {
	ctx.SetContentType("application/json")

	// Check RMQ status
	if r.Ctx.Inst().MessageQueue == nil || !r.Ctx.Inst().MessageQueue.Connected(ctx) {
		return errors.ErrMissingInternalDependency().SetDetail("Emote Processing Service Unavailable")
	}

	// Get actor
	actor, ok := ctx.GetActor()
	if !ok {
		return errors.ErrUnauthorized()
	}

	if !actor.HasPermission(structures.RolePermissionCreateEmote) {
		return errors.ErrInsufficientPrivilege()
	}

	req := &ctx.Request

	var (
		name  string
		tags  []string
		flags structures.EmoteFlag
	)

	// these validations are all "free" as in we can do them before we download the file they try to upload.
	args := &createData{}
	if err := json.Unmarshal(req.Header.Peek("X-Emote-Data"), args); err != nil {
		return errors.ErrInvalidRequest().SetDetail(err.Error())
	}

	if args.Diverged && args.ParentID == nil {
		return errors.ErrInvalidRequest().SetDetail("diverged emote with no parent")
	}

	// Validate: Name
	{
		if !emoteNameRegex.MatchString(args.Name) {
			return errors.ErrInvalidRequest().SetDetail("Bad Emote Name")
		}
		name = args.Name
	}
	// Validate: Flags
	{
		if args.Flags != 0 {
			if utils.BitField.HasBits(int64(args.Flags), int64(structures.EmoteFlagsPrivate)) {
				flags |= structures.EmoteFlagsPrivate
			}
			if utils.BitField.HasBits(int64(args.Flags), int64(structures.EmoteFlagsZeroWidth)) {
				flags |= structures.EmoteFlagsZeroWidth
			}
		}
	}

	// Validate: Tags
	{
		uniqueTags := map[string]bool{}
		if len(args.Tags) > r.Ctx.Config().Limits.Emotes.MaxTags {
			return errors.ErrInvalidRequest().SetDetail(fmt.Sprintf("Too many emote tags %d when the max is %d", len(args.Tags), r.Ctx.Config().Limits.Emotes.MaxTags))
		}

		for _, v := range args.Tags {
			if v == "" {
				continue
			}

			uniqueTags[v] = true
			if !emoteTagRegex.MatchString(v) {
				return errors.ErrInvalidRequest().SetDetail(fmt.Sprintf("Bad Emote Tag '%s'", v))
			}
		}

		tags = make([]string, len(uniqueTags))
		i := 0
		for k := range uniqueTags {
			tags[i] = k
			i++
		}
	}

	id := primitive.NewObjectIDFromTimestamp(time.Now())
	body := req.Body()

	// Create the emote in DB
	eb := structures.NewEmoteBuilder(structures.Emote{
		ID:    id,
		Flags: flags,
	})

	fileType := container.Match(body)
	filekey := fmt.Sprintf("original/emotes/%s.%s", id.Hex(), fileType.Extension)

	version := structures.EmoteVersion{
		Name:        args.Name,
		Description: args.Description,
		InputFile: structures.EmoteFile{
			Name:         "original",
			ContentType:  fileType.MIME.Value,
			Key:          filekey,
			Bucket:       r.Ctx.Config().S3.InternalBucket,
			ACL:          *s3.AclPrivate,
			CacheControl: *s3.DefaultCacheControl,
		},
		ID:        id,
		CreatedAt: id.Timestamp(),
		State: structures.EmoteVersionState{
			Lifecycle: structures.EmoteLifecyclePending,
		},
	}

	if args.ParentID == nil { // new upload
		eb.SetName(name).
			SetOwnerID(actor.ID).
			SetTags(tags, true).
			AddVersion(version)
	} else { // version of existing emote
		// Parse the id of the parent emote
		emotes, err := r.Ctx.Inst().Query.Emotes(ctx, bson.M{"versions.id": *args.ParentID}).Items()
		if err != nil || len(emotes) == 0 {
			return errors.ErrUnknownEmote().SetDetail("Versioning Parent")
		}

		parentEmote := emotes[0]
		eb.Emote = parentEmote

		// Check permissions
		if actor.ID != parentEmote.OwnerID && !actor.HasPermission(structures.RolePermissionEditAnyEmote) && args.Diverged {
			ok := false
			for _, ed := range parentEmote.Owner.Editors {
				if ed.ID == actor.ID && ed.HasPermission(structures.UserEditorPermissionManageOwnedEmotes) {
					ok = true // actor is an editor of emote owner with correct permissions
					break
				}
			}
			if !ok {
				return errors.ErrInsufficientPrivilege()
			}
		}

		// Add as version?
		if args.Diverged {
			// Diverged version;
			// will create a full document with a parent ID
			eb.SetName(parentEmote.Name).
				SetOwnerID(actor.ID).
				SetTags(tags, true).
				AddVersion(version)
		} else {
			eb.AddVersion(version)
		}
	}

	if args.Diverged || args.ParentID == nil {
		if _, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmotes).InsertOne(ctx, eb.Emote); err != nil {
			zap.S().Errorw("mongo, failed to create pending emote in DB",
				"error", err,
			)

			return errors.ErrInternalServerError().SetDetail("Internal Server Error")
		}
	} else {
		if _, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmotes).UpdateByID(ctx, *args.ParentID, eb.Update); err != nil {
			zap.S().Errorw("mongo, failed to add version of emote in DB",
				"error", err,
				"PARENT_EMOTE_ID", args.ParentID.Hex(),
			)

			return errors.ErrInternalServerError().SetDetail("Internal Server Error")
		}
	}

	if err := r.Ctx.Inst().S3.UploadFile(
		ctx,
		&s3manager.UploadInput{
			Body:         bytes.NewBuffer(body),
			Key:          aws.String(filekey),
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
		ID:    id.Hex(),
		Flags: task.TaskFlagALL,
		Input: task.TaskInput{
			Bucket: r.Ctx.Config().S3.InternalBucket,
			Key:    filekey,
		},
		Output: task.TaskOutput{
			Prefix:       path.Join("emotes", id.Hex()),
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
				ID:          id.Hex(),
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

		return errors.ErrInternalServerError().SetDetail("failed to create task")
	}

	return ctx.JSON(rest.Created, &model.Emote{ID: id.Hex()})
}

type createData struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	ParentID    *primitive.ObjectID  `json:"parent_id"`
	Diverged    bool                 `json:"diverged"`
	Tags        []string             `json:"tags"`
	Flags       structures.EmoteFlag `json:"flags"`
}

var (
	emoteNameRegex = regexp.MustCompile(`^[-_A-Za-z():0-9]{2,100}$`)
	emoteTagRegex  = regexp.MustCompile(`^[0-9a-z]{3,30}$`)
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary
