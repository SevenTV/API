package emotes

import (
	"context"
	"fmt"
	"time"

	"github.com/seventv/api/data/events"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"github.com/seventv/compactdisc"
	"github.com/seventv/image-processor/go/task"
	messagequeue "github.com/seventv/message-queue/go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

func listen(gCtx global.Context) {
	epl := &EmoteProcessingListener{gCtx}
	go epl.Listen()
}

type EmoteProcessingListener struct {
	Ctx global.Context
}

func (epl *EmoteProcessingListener) Listen() {
	mq := epl.Ctx.Inst().MessageQueue
	if mq == nil {
		return
	}

	// Results queue
	messages, err := mq.Subscribe(epl.Ctx, messagequeue.Subscription{
		Queue: epl.Ctx.Config().MessageQueue.ImageProcessorResultsQueueName,
		SQS: messagequeue.SubscriptionSQS{
			WaitTimeSeconds: 10,
		},
	})
	if err != nil {
		zap.S().Fatal("EmoteProcessingListener, subscribe to results queue failed")
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
				ctx, cancel := context.WithCancel(epl.Ctx)

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

				if err := epl.HandleResultEvent(ctx, evt); err != nil {
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

	zap.S().Info("stopped emote processing listener")
}

func (epl *EmoteProcessingListener) HandleResultEvent(ctx context.Context, evt task.Result) error {
	// Fetch the emote
	eb := structures.NewEmoteBuilder(structures.Emote{})
	id, err := primitive.ObjectIDFromHex(evt.ID)

	if err != nil {
		return err
	}

	if err := epl.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmotes).FindOne(ctx, bson.M{
		"versions.id": id,
	}).Decode(&eb.Emote); err != nil {
		return err
	}

	imageFiles := []structures.ImageFile{}
	// Iterate through files, append sizes to formats
	for _, file := range evt.ImageOutputs {
		imageFiles = append(imageFiles, structures.ImageFile{
			Name:         file.Name,
			Width:        int32(file.Width),
			Height:       int32(file.Height),
			FrameCount:   int32(file.FrameCount),
			Size:         int64(file.Size),
			ContentType:  file.ContentType,
			SHA3:         file.SHA3,
			Key:          file.Key,
			Bucket:       file.Bucket,
			ACL:          file.ACL,
			CacheControl: file.CacheControl,
		})
	}

	lc := utils.Ternary(evt.State == task.ResultStateSuccess, structures.EmoteLifecycleLive, structures.EmoteLifecycleFailed)
	ver, verIndex := eb.Emote.GetVersion(id)

	if evt.State == task.ResultStateFailed {
		ver.State.Error = evt.Message
	}

	ver.Animated = int32(evt.ImageInput.FrameCount) > 1
	ver.State.Lifecycle = lc
	ver.StartedAt = evt.StartedAt
	ver.CompletedAt = evt.FinishedAt
	ver.InputFile = structures.ImageFile{
		Name:         utils.Ternary(evt.ImageInput.Name != "", evt.ImageInput.Name, ver.InputFile.Name),
		Key:          utils.Ternary(evt.ImageInput.Key != "", evt.ImageInput.Key, ver.InputFile.Key),
		Bucket:       utils.Ternary(evt.ImageInput.Bucket != "", evt.ImageInput.Bucket, ver.InputFile.Bucket),
		ACL:          utils.Ternary(evt.ImageInput.ACL != "", evt.ImageInput.ACL, ver.InputFile.ACL),
		CacheControl: utils.Ternary(evt.ImageInput.CacheControl != "", evt.ImageInput.CacheControl, ver.InputFile.CacheControl),
		ContentType:  evt.ImageInput.ContentType,
		FrameCount:   int32(evt.ImageInput.FrameCount),
		Size:         int64(evt.ImageInput.Size),
		Height:       int32(evt.ImageInput.Height),
		Width:        int32(evt.ImageInput.Width),
		SHA3:         evt.ImageInput.SHA3,
	}
	ver.ImageFiles = imageFiles
	ver.ArchiveFile = structures.ImageFile{
		Name:         evt.ArchiveOutput.Name,
		Size:         int64(evt.ArchiveOutput.Size),
		ContentType:  "application/zip",
		SHA3:         evt.ArchiveOutput.SHA3,
		Key:          evt.ArchiveOutput.Key,
		Bucket:       evt.ArchiveOutput.Bucket,
		ACL:          evt.ArchiveOutput.ACL,
		CacheControl: evt.ArchiveOutput.CacheControl,
	}

	eb.Update.Set(fmt.Sprintf("versions.%d.animated", verIndex), ver.Animated)
	eb.Update.Set(fmt.Sprintf("versions.%d.state.lifecycle", verIndex), ver.State.Lifecycle)
	eb.Update.Set(fmt.Sprintf("versions.%d.state.error", verIndex), ver.State.Error)
	eb.Update.Set(fmt.Sprintf("versions.%d.started_at", verIndex), ver.StartedAt)
	eb.Update.Set(fmt.Sprintf("versions.%d.completed_at", verIndex), ver.CompletedAt)
	eb.Update.Set(fmt.Sprintf("versions.%d.input_file", verIndex), ver.InputFile)
	eb.Update.Set(fmt.Sprintf("versions.%d.image_files", verIndex), ver.ImageFiles)
	eb.Update.Set(fmt.Sprintf("versions.%d.archive_file", verIndex), ver.ArchiveFile)

	// Update database
	_, err = epl.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmotes).UpdateOne(ctx, bson.M{
		"versions.id": id,
	}, eb.Update)

	if err == nil {
		err = epl.Ctx.Inst().Redis.RawClient().Publish(ctx, fmt.Sprintf("events:sub:emotes:%s", id.Hex()), "1").Err()
	}

	metadata := TaskMetadata{}
	_ = json.Unmarshal(evt.Metadata, &metadata)

	if err == nil {
		// Write re-processing log?
		if metadata.Reprocessed.Done {
			ab := structures.NewAuditLogBuilder(structures.AuditLog{Changes: []*structures.AuditLogChange{}}).
				SetActor(metadata.Reprocessed.Actor).
				SetTargetID(id).
				SetTargetKind(structures.ObjectKindEmote).
				SetKind(structures.AuditLogKindProcessEmote)
			if _, err = epl.Ctx.Inst().Mongo.Collection(mongo.CollectionNameAuditLogs).InsertOne(ctx, ab.AuditLog); err != nil {
				zap.S().Errorw("failed to write an audit log about the reprocessing of an emote",
					"error", err,
					"EMOTE_ID", id,
					"ACTOR_ID", metadata.Reprocessed.Actor,
				)
			}
		} else {
			// Send an Event API update about the emote's lifecycle state
			fields := []events.ChangeField{
				{
					Key:      "lifecycle",
					Type:     events.ChangeFieldTypeNumber,
					OldValue: structures.EmoteLifecycleProcessing,
					Value:    ver.State.Lifecycle,
				},
				{
					Key:    "versions",
					Index:  utils.PointerOf(int32(verIndex)),
					Nested: true,
					Value: []events.ChangeField{{
						Key:      "lifecycle",
						Type:     events.ChangeFieldTypeNumber,
						OldValue: structures.EmoteLifecycleProcessing,
						Value:    ver.State.Lifecycle,
					}},
				},
			}

			emoteOwner, _ := epl.Ctx.Inst().Loaders.UserByID().Load(eb.Emote.OwnerID)

			if ver.State.Lifecycle == structures.EmoteLifecycleFailed {
				fields = append(fields, events.ChangeField{
					Key:    "versions",
					Index:  utils.PointerOf(int32(verIndex)),
					Type:   events.ChangeFieldTypeObject,
					Nested: true,
					Value: []events.ChangeField{{
						Key:      "error",
						Type:     events.ChangeFieldTypeString,
						OldValue: nil,
						Value:    &ver.State.Error,
					}},
				})
			} else {
				// Create a mod request for the new emote to be approved
				mb := structures.NewMessageBuilder(structures.Message[structures.MessageDataModRequest]{}).
					SetKind(structures.MessageKindModRequest).
					SetAuthorID(eb.Emote.OwnerID).
					SetTimestamp(time.Now()).
					SetData(structures.MessageDataModRequest{
						TargetKind: structures.ObjectKindEmote,
						TargetID:   id,
					})
				if err = epl.Ctx.Inst().Mutate.SendModRequestMessage(ctx, mb); err != nil {
					zap.S().Errorw("failed to send mod request message for new emote",
						"error", err,
						"EMOTE_ID", id,
						"ACTOR_ID", eb.Emote.OwnerID,
					)
				}

				// Send a message on discord
				_, _ = epl.Ctx.Inst().CD.SendMessage("activity_feed", compactdisc.MessageSend{
					Content: fmt.Sprintf(
						"**[activity]** emote created: [%s](%s) by [%s](%s)",
						eb.Emote.Name, eb.Emote.WebURL(epl.Ctx.Config().WebsiteURL),
						emoteOwner.DisplayName, emoteOwner.WebURL(epl.Ctx.Config().WebsiteURL),
					),
				}, true)
			}

			_ = epl.Ctx.Inst().Events.Dispatch(ctx, events.EventTypeUpdateEmote, events.ChangeMap{
				ID:      eb.Emote.ID,
				Kind:    structures.ObjectKindEmote,
				Actor:   epl.Ctx.Inst().Modelizer.User(emoteOwner),
				Updated: fields,
			}, events.EventCondition{}.SetObjectID(eb.Emote.ID))
		}
	}

	return err
}

type EmoteJobEvent struct {
	JobID     primitive.ObjectID
	Type      EmoteJobEventType
	Timestamp time.Time
}

type EmoteJobEventType string

const (
	EmoteJobEventTypeStarted            EmoteJobEventType = "started"
	EmoteJobEventTypeDownloaded         EmoteJobEventType = "downloaded"
	EmoteJobEventTypeStageOne           EmoteJobEventType = "stage-one"
	EmoteJobEventTypeStageOneComplete   EmoteJobEventType = "stage-one-complete"
	EmoteJobEventTypeStageTwo           EmoteJobEventType = "stage-two"
	EmoteJobEventTypeStageTwoComplete   EmoteJobEventType = "stage-two-complete"
	EmoteJobEventTypeStageThree         EmoteJobEventType = "stage-three"
	EmoteJobEventTypeStageThreeComplete EmoteJobEventType = "stage-three-complete"
	EmoteJobEventTypeCompleted          EmoteJobEventType = "completed"
	EmoteJobEventTypeCleaned            EmoteJobEventType = "cleaned"
)

type EmoteResultEvent struct {
	JobID   primitive.ObjectID `json:"job_id"`
	Success bool               `json:"success"`
	Files   []EmoteResultFile  `json:"files"`
	Error   string             `json:"error"`
}

type EmoteResultFile struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	Animated    bool   `json:"animated"`
	TimeTaken   int    `json:"time_taken"`
	Width       int32  `json:"width"`
	Height      int32  `json:"height"`
}

type TaskMetadata struct {
	Reprocessed TaskMetadataReprocessed `json:"reprocessed"`
}

type TaskMetadataReprocessed struct {
	Done  bool               `json:"done"`
	Actor primitive.ObjectID `json:"actor"`
}
