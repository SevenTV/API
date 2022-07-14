package auth

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/api/internal/rest/v3/middleware"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"google.golang.org/api/youtube/v3"
)

type youtubeRoot struct {
	Ctx global.Context
}

func newYouTube(gctx global.Context) rest.Route {
	return &youtubeRoot{gctx}
}

func (r *youtubeRoot) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/youtube",
		Method: rest.GET,
		Children: []rest.Route{
			newYouTubeRequest(r.Ctx),
			newYouTubeVerify(r.Ctx),
		},
	}
}

func (r *youtubeRoot) Handler(ctx *rest.Ctx) rest.APIError {
	return errors.ErrUnknownRoute()
}

// Request Verification

type youtubeRequest struct {
	Ctx global.Context
}

func newYouTubeRequest(gctx global.Context) rest.Route {
	return &youtubeRequest{gctx}
}

func (r *youtubeRequest) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/request-verification",
		Method: rest.GET,
		Middleware: []rest.Middleware{
			middleware.Auth(r.Ctx),
		},
	}
}

func (r *youtubeRequest) Handler(ctx *rest.Ctx) rest.APIError {
	if r.Ctx.Inst().YouTube == nil {
		return errors.ErrMissingInternalDependency().SetDetail("YouTube API is not setup on this server")
	}

	channelID := utils.B2S(ctx.QueryArgs().Peek("channel_id"))
	if channelID == "" {
		return errors.ErrInvalidRequest().SetDetail("channel_id is a required query parameter")
	}

	// getters attempt to retrieve a youtube channel by different parameters
	getters := []func(c context.Context, v string) (*youtube.Channel, error){
		r.Ctx.Inst().YouTube.GetChannelByID,
		r.Ctx.Inst().YouTube.GetChannelByUsername,
	}

	var (
		channel *youtube.Channel
		err     error
	)

	for _, f := range getters {
		channel, err = f(ctx, channelID)
		if err == nil {
			break
		}
	}

	if channel == nil { // no channel was found with the id passed by the user
		return errors.ErrUnknownUser().SetDetail("Unable to find YouTube channel")
	}

	if count, _ := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).CountDocuments(ctx, bson.M{
		"connections.id": channel.Id,
	}); count > 0 {
		return errors.ErrInsufficientPrivilege().SetDetail("This channel is already bound to another user")
	}

	// Generate a random string that will be used to verify the requester owns the channel
	tokenBytes, err := utils.GenerateRandomBytes(24)
	if err != nil {
		zap.S().Errorw("youtube, couldn't generate verification token", "error", err)
		return errors.ErrInternalServerError()
	}

	token := hex.EncodeToString(tokenBytes)

	actor, ok := ctx.GetActor()

	if !ok {
		return errors.ErrUnauthorized()
	}

	// Store this token in state
	// it can now be used to claim ownership of the channel
	if err = r.Ctx.Inst().Redis.SetEX(
		ctx,
		r.Ctx.Inst().Redis.ComposeKey("rest-v3", actor.ID.Hex(), channelID),
		token,
		time.Hour*2,
	); err != nil {
		zap.S().Errorw("youtube, failed to save state of verification token")
		return errors.ErrInternalServerError()
	}

	result := verificationRequestResult{
		Token:              token,
		VerificationString: fmt.Sprintf(`[7TV VERIFY]:"%s"`, token),
		ManageChannelURL:   fmt.Sprintf("https://studio.youtube.com/channel/%s/editing/details", channel.Id),
		ChannelID:          channel.Id,
		Channel:            channel,
	}

	return ctx.JSON(rest.OK, result)
}

type verificationRequestResult struct {
	Token              string           `json:"token"`
	VerificationString string           `json:"verification_string"`
	ManageChannelURL   string           `json:"manage_channel_url"`
	ChannelID          string           `json:"channel_id"`
	Channel            *youtube.Channel `json:"channel"`
}
