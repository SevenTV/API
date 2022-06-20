package emotes

import (
	"fmt"
	"sort"
	"time"

	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	// rmq := epl.Ctx.Inst().Rmq
	// if rmq == nil { // RMQ not set up; ignore
	// 	return
	// }

	// // Update queue
	// ch1, err := rmq.Subscribe(epl.Ctx.Config().Rmq.UpdateQueueName)
	// if err != nil {
	// 	logrus.WithError(err).Fatalf("EmoteProcessingListener, rmq, subscribe to update queue failed")
	// }

	// // Results queue
	// ch2, err := rmq.Subscribe(epl.Ctx.Config().Rmq.ResultQueueName)
	// if err != nil {
	// 	logrus.WithError(err).Fatal("EmoteProcessingListener, rmq, subscribe to results queue failed")
	// }

	// wg := sync.WaitGroup{}
	// wg.Add(2)
	// go func() {
	// 	defer wg.Done()

	// 	var msg amqp.Delivery
	// 	for {
	// 		select {
	// 		case msg = <-ch1:
	// 			evt := &EmoteJobEvent{}
	// 			if err = json.Unmarshal(msg.Body, evt); err != nil {
	// 				logrus.WithError(err).Error("EmoteProcessingListener, failed to decode emote processing event")
	// 				return
	// 			}

	// 			if err = epl.HandleUpdateEvent(evt); err != nil {
	// 				logrus.WithError(err).Error("EmoteProcessingListener, failed to handle event")
	// 			}
	// 			_ = msg.Ack(false)
	// 		case <-epl.Ctx.Done():
	// 			return
	// 		}
	// 	}
	// }()

	// go func() {
	// 	defer wg.Done()

	// 	var msg amqp.Delivery
	// 	for {
	// 		select {
	// 		case msg = <-ch2:
	// 			evt := &EmoteResultEvent{}
	// 			if err = json.Unmarshal(msg.Body, evt); err != nil {
	// 				logrus.WithError(err).Error("EmoteProcessingListener, failed to decode emote result event")
	// 				return
	// 			}

	// 			if err = epl.HandleResultEvent(evt); err != nil {
	// 				logrus.WithError(err).Error("EmoteProcessingListener, failed to handle event")
	// 			}
	// 			_ = msg.Ack(false)
	// 		case <-epl.Ctx.Done():
	// 			return
	// 		}
	// 	}
	// }()

	// wg.Wait()
	// logrus.Info("stopped emote processing listener")
}

func (epl *EmoteProcessingListener) HandleUpdateEvent(evt *EmoteJobEvent) error {
	// Fetch the emote
	eb := structures.NewEmoteBuilder(structures.Emote{})
	if err := epl.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmotes).FindOne(epl.Ctx, bson.M{
		"versions.id": evt.JobID,
	}).Decode(eb.Emote); err != nil {
		return err
	}

	// Store the state in redis
	epl.Ctx.Inst().Redis.RawClient().Set(epl.Ctx, fmt.Sprintf("emote-processing:%s:status", evt.JobID), evt.Type, time.Minute)

	logf := zap.S().With("emote_id", evt.JobID)
	switch evt.Type {
	case EmoteJobEventTypeStarted:
		ver, i := eb.Emote.GetVersion(evt.JobID)
		if !ver.ID.IsZero() {
			eb.Update.Set(fmt.Sprintf("versions.%d.state.lifecycle", i), structures.EmoteLifecycleProcessing)
		}
		logf.Info("Emote Processing Started")
	case EmoteJobEventTypeCompleted:
		logf.Info("Emote Processing Complete")
		ver, i := eb.Emote.GetVersion(evt.JobID)
		if !ver.ID.IsZero() {
			logf.Error("couldn't find version of the emote for this job")
			break
		}
		eb.Update.Set(fmt.Sprintf("versions.%d.state.lifecycle", i), structures.EmoteLifecycleLive)
	default:
		logf.Infof("Emote Processing Status: %s", evt.Type)
	}

	// Update the emote in DB if status was updated
	if len(eb.Update) > 0 {
		if _, err := epl.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmotes).UpdateOne(epl.Ctx, bson.M{"versions.id": eb.Emote.ID}, eb.Update); err != nil {
			return err
		}
	}
	epl.Ctx.Inst().Redis.RawClient().Publish(epl.Ctx, fmt.Sprintf("7tv-events:sub:emotes:%s", eb.Emote.ID.Hex()), "1")

	return nil
}

func (epl *EmoteProcessingListener) HandleResultEvent(evt *EmoteResultEvent) error {
	// Fetch the emote
	eb := structures.NewEmoteBuilder(structures.Emote{})
	if err := epl.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmotes).FindOne(epl.Ctx, bson.M{
		"versions.id": evt.JobID,
	}).Decode(eb.Emote); err != nil {
		return err
	}

	// Map formats
	formats := make(map[structures.EmoteFormatName]*structures.EmoteFormat)

	// Iterate through files, append sizes to formats
	for _, file := range evt.Files {
		cType := structures.EmoteFormatName(file.ContentType)
		format := formats[cType]
		if format == nil {
			format = &structures.EmoteFormat{
				Name:  cType,
				Files: []structures.EmoteFile{},
			}
			formats[cType] = format
		}

		format.Files = append(format.Files, structures.EmoteFile{
			Name:           file.Name,
			Width:          file.Width,
			Height:         file.Height,
			Animated:       file.Animated,
			ProcessingTime: int64(file.TimeTaken),
			Length:         file.Size,
		})
		// Sort the files, by width
		sort.Slice(format.Files, func(i, j int) bool {
			a := format.Files[i]
			b := format.Files[j]
			return b.Width > a.Width
		})
	}

	// Create formats list to set in DB
	formatList := []structures.EmoteFormat{}
	for _, format := range formats {
		if format == nil {
			continue
		}
		formatList = append(formatList, *format)
	}

	lc := utils.Ternary(evt.Success, structures.EmoteLifecycleLive, structures.EmoteLifecycleFailed)
	ver, verIndex := eb.Emote.GetVersion(evt.JobID)
	ver.State.Lifecycle = lc
	ver.Formats = formatList
	eb.Update.Set(fmt.Sprintf("versions.%d.state.lifecycle", verIndex), lc)
	eb.Update.Set(fmt.Sprintf("versions.%d.formats", verIndex), formatList)

	// Update database
	_, err := epl.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmotes).UpdateOne(epl.Ctx, bson.M{
		"versions.id": evt.JobID,
	}, eb.Update)

	epl.Ctx.Inst().Redis.RawClient().Publish(epl.Ctx, fmt.Sprintf("7tv-events:sub:emotes:%s", evt.JobID.Hex()), "1")
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
