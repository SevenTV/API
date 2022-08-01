package emotes

import (
	"bytes"
	"fmt"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/h2non/filetype/matchers"
	jsoniter "github.com/json-iterator/go"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/middleware"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/api/internal/rest/v3/model"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/query"
	"github.com/seventv/common/svc/s3"
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
			middleware.Auth(r.Ctx, true),
			middleware.RateLimit(r.Ctx, "CreateEmote", r.Ctx.Config().Limits.Buckets.ImageProcessing),
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
	done := r.Ctx.Inst().Limiter.AwaitMutation(ctx)
	defer done()

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

	reqs, err := r.Ctx.Inst().Query.ModRequestMessages(ctx, query.ModRequestMessagesQueryOptions{
		Actor: &actor,
		Targets: map[structures.ObjectKind]bool{
			structures.ObjectKindEmote: true,
		},
		Filter: bson.M{
			"author_id": actor.ID,
		},
		SkipPermissionCheck: true,
	}).Items()
	if err != nil && !errors.Compare(err, errors.ErrNoItems()) {
		return errors.ErrInternalServerError().SetDetail("Unable to evaluate active mod requests")
	}

	emoteIDs := []primitive.ObjectID{}

	for _, re := range reqs {
		msg, err := structures.ConvertMessage[structures.MessageDataModRequest](re)
		if err == nil {
			emoteIDs = append(emoteIDs, msg.Data.TargetID)
		}
	}

	reqLimit := r.Ctx.Config().Limits.Quota.MaxActiveModRequests
	if count, _ := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmotes).CountDocuments(ctx, bson.M{
		"versions.id":              bson.M{"$in": emoteIDs},
		"versions.state.lifecycle": structures.EmoteLifecycleLive,
	}); count >= reqLimit {
		return errors.ErrRateLimited().SetDetail("You have too many emotes pending approval!")
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
		ID:          id,
		Flags:       flags,
		ChildrenIDs: []primitive.ObjectID{},
	})

	fileType := container.Match(body)
	switch fileType {
	case container.TypeAvif:
	case matchers.TypeWebp:
	case matchers.TypeGif:
	case matchers.TypePng:
	case matchers.TypeTiff:
	case matchers.TypeJpeg:
	case matchers.TypeWebm:
	case matchers.TypeMp4:
	case matchers.TypeFlv:
	case matchers.TypeAvi:
	case matchers.TypeMov:
	default:
		return errors.ErrInvalidRequest().SetDetail(fmt.Sprintf("Bad emote upload type '%s'", fileType.MIME.Value))
	}

	filekey := r.Ctx.Inst().S3.ComposeKey("emote", id.Hex(), fmt.Sprintf("raw.%s", fileType.Extension))

	version := structures.EmoteVersion{
		Name:        args.Name,
		Description: args.Description,
		ImageFiles:  []structures.EmoteFile{},
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
		parentEmote, err := r.Ctx.Inst().Query.Emotes(ctx, bson.M{"versions.id": *args.ParentID}).First()
		if err != nil {
			return errors.ErrUnknownEmote().SetDetail("Versioning Parent")
		}

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

		ver, _ := parentEmote.GetVersion(*args.ParentID)
		if ver.IsUnavailable() {
			return errors.ErrInsufficientPrivilege().SetDetail("Parent is unavailable")
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

			return errors.ErrInternalServerError().SetDetail("Couldn't define initial record")
		}
	} else {
		if _, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmotes).UpdateByID(ctx, *args.ParentID, eb.Update); err != nil {
			zap.S().Errorw("mongo, failed to add version of emote in DB",
				"error", err,
				"PARENT_EMOTE_ID", args.ParentID.Hex(),
			)

			return errors.ErrInternalServerError().SetDetail("Couldn't define initial version")
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
		ID: id.Hex(),
		Flags: task.TaskFlagAVIF |
			task.TaskFlagAVIF_STATIC |
			task.TaskFlagGIF |
			task.TaskFlagPNG |
			task.TaskFlagPNG_STATIC |
			task.TaskFlagWEBP |
			task.TaskFlagWEBP_STATIC,
		Input: task.TaskInput{
			Bucket: r.Ctx.Config().S3.InternalBucket,
			Key:    filekey,
		},
		Output: task.TaskOutput{
			Prefix:               r.Ctx.Inst().S3.ComposeKey("emote", id.Hex()),
			Bucket:               r.Ctx.Config().S3.PublicBucket,
			ACL:                  *s3.AclPublicRead,
			CacheControl:         *s3.DefaultCacheControl,
			ExcludeFileExtension: true,
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

	// Create a new audit log
	ab := structures.NewAuditLogBuilder(structures.AuditLog{Changes: []*structures.AuditLogChange{}}).
		SetActor(eb.Emote.OwnerID).
		SetTargetID(id).
		SetTargetKind(structures.ObjectKindEmote).
		SetKind(structures.AuditLogKindCreateEmote)
	if _, err = r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameAuditLogs).InsertOne(ctx, ab.AuditLog); err != nil {
		zap.S().Errorw("failed to create an audit log about the creation of an emote",
			"error", err,
			"EMOTE_ID", id,
			"ACTOR_ID", actor.ID,
		)
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
