package auth

import (
	"fmt"
	"regexp"
	"time"

	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/api/internal/rest/v3/middleware"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"google.golang.org/api/youtube/v3"
)

type youtubeVerify struct {
	Ctx global.Context
}

func newYouTubeVerify(gctx global.Context) rest.Route {
	return &youtubeVerify{gctx}
}

func (r *youtubeVerify) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/verify",
		Method: rest.GET,
		Middleware: []rest.Middleware{
			middleware.Auth(r.Ctx),
		},
	}
}
func (r *youtubeVerify) Handler(ctx *rest.Ctx) rest.APIError {
	if r.Ctx.Inst().YouTube == nil {
		return errors.ErrMissingInternalDependency().SetDetail("YouTube API is not setup on this server")
	}

	channelID := utils.B2S(ctx.QueryArgs().Peek("channel_id"))
	if channelID == "" {
		return errors.ErrInvalidRequest().SetDetail("channel_id is a required query parameter")
	}

	// Retrieve session
	actor, ok := ctx.GetActor()
	if !ok {
		return errors.ErrUnauthorized()
	}

	rkey := r.Ctx.Inst().Redis.ComposeKey("rest-v3", actor.ID.Hex(), channelID)
	tok, err := r.Ctx.Inst().Redis.Get(ctx, rkey)

	if err != nil {
		if err == redis.Nil {
			return errors.ErrInsufficientPrivilege().SetDetail("There is no verification flow for that channel, or it expired")
		}

		return errors.ErrInternalServerError()
	}

	// Re-fetch the channel
	channel, err := r.Ctx.Inst().YouTube.GetChannelByID(ctx, channelID)
	if err != nil {
		return errors.ErrUnknownUser().SetDetail("The channel associated with this verification flow is not available")
	}

	// Form a regex to parse out the verification token
	regex, err := regexp.Compile(fmt.Sprintf(`\[7TV VERIFY\]:"(%s?)"`, tok))
	if err != nil {
		return errors.ErrInternalServerError()
	}

	// Check for a match with the description
	if ok := regex.MatchString(channel.Snippet.Description); !ok {
		return errors.ErrInvalidRequest().SetDetail("The token was not found in the channel's description. Please try again")
	}

	// Now we know the user controls the channel
	// Let's create a youtube connection
	tw, _, _ := actor.Connections.Twitch()

	ucb := structures.NewUserConnectionBuilder(structures.UserConnection[structures.UserConnectionDataYoutube]{
		ID:         channel.Id,
		Platform:   structures.UserConnectionPlatformYouTube,
		LinkedAt:   time.Now(),
		EmoteSlots: 250,
		EmoteSetID: tw.EmoteSetID, // infer set from twitch conn if it exists
	}).SetID(channel.Id).
		SetPlatform(structures.UserConnectionPlatformYouTube).
		SetLinkedAt(time.Now()).
		SetActiveEmoteSet(tw.EmoteSetID).
		SetData(structures.UserConnectionDataYoutube{
			ID:          channelID,
			Title:       channel.Snippet.Title,
			Description: channel.Snippet.Description,
			ViewCount:   int64(channel.Statistics.ViewCount),
			SubCount:    int64(channel.Statistics.SubscriberCount),
		})

	ub := structures.NewUserBuilder(*actor)
	ub.AddConnection(ucb.UserConnection.ToRaw())

	// Write to DB
	if _, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).UpdateOne(ctx, bson.M{
		"_id": actor.ID,
	}, ub.Update); err != nil {
		zap.S().Errorw("mongo, failed to write update of user verifying their youtube channel",
			"error", err,
		)

		return errors.ErrInternalServerError()
	}

	// Clear redis
	_, _ = r.Ctx.Inst().Redis.Del(ctx, rkey)

	return ctx.JSON(rest.OK, verifyResult{
		Channel:   channel,
		ChannelID: channel.Id,
		Verified:  true,
		UserID:    actor.ID.Hex(),
	})
}

type verifyResult struct {
	Channel   *youtube.Channel `json:"channel"`
	ChannelID string           `json:"channel_id"`
	Verified  bool             `json:"verified"`
	UserID    string           `json:"user_id"`
}
