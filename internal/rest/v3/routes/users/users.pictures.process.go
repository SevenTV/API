package users

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/seventv/api/internal/events"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/image-processor/go/task"
	messagequeue "github.com/seventv/message-queue/go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	aid, err := primitive.ObjectIDFromHex(evt.ID)
	if err != nil {
		l.Errorw("failed to parse task id")
		return err
	}

	var uid primitive.ObjectID
	if err := json.Unmarshal(evt.Metadata, &uid); err != nil {
		l.Errorw("failed to parse metadata")
		return err
	}

	// Find the user that triggered this job
	// Fetch the full data about the actor
	actor, err := ppl.Ctx.Inst().Loaders.UserByID().Load(uid)
	if err != nil {
		l.Errorw("failed to fetch actor")
		return err
	}

	if actor.Avatar != nil && actor.Avatar.ID != aid && actor.Avatar.PendingID != nil && *actor.Avatar.PendingID != aid {
		l.Error("avatar was changed while processing")
		return nil
	}

	inputFile := jobImageToStructImage(evt.ImageInput)

	imagesFiles := make([]structures.ImageFile, len(evt.ImageOutputs))
	for i, im := range evt.ImageOutputs {
		imagesFiles[i] = jobImageToStructImage(im)
	}

	if _, err := ppl.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).UpdateOne(ctx, bson.M{
		"_id": uid,
	}, bson.M{
		"$set": bson.M{"avatar": structures.UserAvatar{
			ID:         aid,
			InputFile:  inputFile,
			ImageFiles: imagesFiles,
		}},
	}); err != nil {
		l.Errorw("failed to update user avatar id", "error", err)
		return err
	}

	events.Publish(ppl.Ctx, "users", actor.ID)

	return nil
}

func jobImageToStructImage(im task.ResultFile) structures.ImageFile {
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
