package youtube

import (
	"context"

	"github.com/seventv/common/errors"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type Instance interface {
	GetChannelByID(ctx context.Context, id string) (*youtube.Channel, error)
	GetChannelByUsername(ctx context.Context, name string) (*youtube.Channel, error)
}

type youtubeInst struct {
	api *youtube.Service
}

func New(ctx context.Context, opt YouTubeOptions) (Instance, error) {
	ytapi, err := youtube.NewService(ctx, option.WithAPIKey(opt.APIKey))
	if err != nil {
		return nil, err
	}

	return &youtubeInst{
		api: ytapi,
	}, nil
}

type YouTubeOptions struct {
	APIKey string
}

func (inst *youtubeInst) GetChannelByID(ctx context.Context, id string) (*youtube.Channel, error) {
	res, err := inst.api.Channels.List([]string{"snippet", "statistics"}).Id(id).Context(ctx).Do()
	if err != nil {
		return nil, errors.ErrInternalServerError()
	}

	if len(res.Items) == 0 {
		return nil, errors.ErrNoItems()
	}

	channel := res.Items[0]

	return channel, nil
}

func (inst *youtubeInst) GetChannelByUsername(ctx context.Context, name string) (*youtube.Channel, error) {
	res, err := inst.api.Channels.List([]string{"snippet", "statistics"}).ForUsername(name).Context(ctx).Do()
	if err != nil {
		return nil, errors.ErrInternalServerError()
	}

	if len(res.Items) == 0 {
		return nil, errors.ErrNoItems()
	}

	channel := res.Items[0]

	return channel, nil
}
