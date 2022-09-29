package users

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	s3_opts "github.com/aws/aws-sdk-go/service/s3"
	"github.com/seventv/api/internal/events"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/svc/s3"
	"github.com/seventv/common/utils"
	"github.com/seventv/image-processor/go/task"
	messagequeue "github.com/seventv/message-queue/go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

func userPictureListener(gCtx global.Context) {
	ppl := &PictureProcessingListener{gCtx}
	go ppl.Listen()
}

type PictureProcessingListener struct {
	Ctx global.Context
}

func (ppl *PictureProcessingListener) Listen() {
	mq := ppl.Ctx.Inst().MessageQueue
	if mq == nil {
		return
	}

	// Results queue
	messages, err := mq.Subscribe(ppl.Ctx, messagequeue.Subscription{
		Queue: ppl.Ctx.Config().MessageQueue.ImageProcessorUserPicturesResultsQueueName,
		SQS: messagequeue.SubscriptionSQS{
			WaitTimeSeconds: 10,
		},
	})
	if err != nil {
		zap.S().Fatal("PictureProcessingListener, subscribe to results queue failed")
	}

	evt := task.Result{}

	for msg := range messages {
		if msg.Headers().ContentType() == "application/json" {
			if err := json.Unmarshal(msg.Body(), &evt); err != nil {
				zap.S().Errorw("bad message type from queue",
					"msg", msg,
				)

				continue
			}

			go func(msg *messagequeue.IncomingMessage) {
				tick := time.NewTicker(time.Second * 10)
				ctx, cancel := context.WithCancel(ppl.Ctx)

				defer cancel()
				defer tick.Stop()

				go func() {
					for range tick.C {
						if err := msg.Extend(context.Background(), time.Second*30); err != nil && err != messagequeue.ErrUnimplemented {
							zap.S().Errorw("failed to extend message",
								"error", err,
							)
							cancel()

							return
						}
					}
				}()

				if err := ppl.HandleResultEvent(ctx, evt); err != nil {
					zap.S().Errorw("failed to handle result",
						"error", multierr.Append(err, msg.Nack(context.Background())),
					)
				} else {
					if err = msg.Ack(ctx); err != nil {
						zap.S().Errorw("failed to ack message",
							"error", err,
						)
					}
				}
			}(msg)
		} else {
			zap.S().Warnw("bad message type from queue",
				"msg", msg,
			)
			if err = msg.Nack(context.Background()); err != nil {
				zap.S().Errorw("failed to nack message",
					"error", err,
				)
			}
		}
	}

	zap.S().Info("stopped user picture processing listener")
}

func (ppl *PictureProcessingListener) HandleResultEvent(ctx context.Context, evt task.Result) error {
	if len(evt.ImageOutputs) == 0 {
		return fmt.Errorf("no image outputs")
	}

	l := zap.S().Named("profile picture processing").With("task_id", evt.ID)

	oid, err := primitive.ObjectIDFromHex(evt.ID)
	if err != nil {
		l.Errorw("failed to parse task id")
		return err
	}

	var (
		img    task.ResultImage
		static task.ResultImage
	)

	// Find the user that triggered this job
	var actor structures.User
	if err = ppl.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).FindOne(ctx, bson.M{
		"avatar.pending_id": oid,
	}, options.FindOne().SetProjection(bson.M{"_id": 1})).Decode(&actor); err != nil {
		return err
	}

	// Fetch the full data about the actor
	actor, err = ppl.Ctx.Inst().Loaders.UserByID().Load(actor.ID)
	if err != nil {
		l.Errorw("failed to fetch actor")
	}

	allowAnim := actor.HasPermission(structures.RolePermissionFeatureProfilePictureAnimation)

	for _, im := range evt.ImageOutputs {
		if im.ContentType != "image/webp" {
			continue
		}

		if im.FrameCount > 1 {
			img = im
		} else {
			static = im
		}
	}

	if img.Key == "" || !allowAnim {
		img = static
	}

	inputKey := ppl.Ctx.Config().S3.PublicBucket + "/" + utils.Ternary(allowAnim, img.Key, static.Key)
	outputKey := strings.TrimSuffix(img.Key, path.Base(inputKey)) + "/" + evt.ID

	if err := ppl.Ctx.Inst().S3.CopyFile(ctx, &s3_opts.CopyObjectInput{
		ACL:          aws.String(*s3.AclPublicRead),
		Bucket:       &ppl.Ctx.Config().S3.PublicBucket,
		CacheControl: s3.DefaultCacheControl,
		CopySource:   aws.String(inputKey),
		Key:          aws.String(outputKey),
	}); err != nil {
		l.Errorw("failed to copy object to correct location for user profile picture",
			"error", "err",
			"task_id", evt.ID,
		)

		return err
	}

	inputFile := jobImageToStructImage(evt.ImageInput)

	imagesFiles := make([]structures.ImageFile, len(evt.ImageOutputs))
	for i, im := range evt.ImageOutputs {
		imagesFiles[i] = jobImageToStructImage(im)
	}

	if _, err := ppl.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).UpdateOne(ctx, bson.M{
		"avatar.pending_id": evt.ID,
	}, bson.M{
		"$set": bson.M{"avatar": structures.UserAvatar{
			ID:         oid,
			InputFile:  inputFile,
			ImageFiles: imagesFiles,
		}},
		"$unset": bson.M{"avatar.pending_id": 1},
	}); err != nil {
		l.Errorw("failed to update user avatar id", "error", err)
	}

	events.Publish(ppl.Ctx, "users", actor.ID)

	return nil
}

func jobImageToStructImage(im task.ResultImage) structures.ImageFile {
	return structures.ImageFile{
		Name:         im.Name,
		Key:          im.Key,
		Bucket:       im.Bucket,
		ACL:          im.ACL,
		CacheControl: im.CacheControl,
		ContentType:  im.ContentType,
		FrameCount:   int32(im.FrameCount),
		Size:         int64(im.Size),
		Width:        int32(im.Width),
		Height:       int32(im.Height),
		SHA3:         im.SHA3,
	}
}
