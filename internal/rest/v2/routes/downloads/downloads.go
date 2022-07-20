package downloads

import (
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/middleware"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/common/errors"
)

type Route struct {
	Ctx global.Context
}

func New(gctx global.Context) rest.Route {
	return &Route{gctx}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/webext",
		Method: rest.GET,
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 600, nil),
		},
	}
}

// Downloads
// @Summary Get Downloads
// @Description Lists downloadable extensions and apps
// @Produce json
// @Success 200 {object} DownloadsResult
// @Router /webext [get]
func (r *Route) Handler(ctx *rest.Ctx) errors.APIError {
	platforms := []Platform{ // TODO: infer this data from a config
		{
			ID:         "chrome",
			VersionTag: "2.2.2",
			New:        false,
		},
		{
			ID:         "firefox",
			VersionTag: "2.2.2",
			New:        false,
		},
		{
			ID:         "chatterino",
			VersionTag: "7.3.5",
			New:        false,
		},
		{
			ID:         "mobile",
			VersionTag: "MOBILE",
			New:        false,
			Variants: []PlatformVariant{
				{
					Name:        "Chatsen",
					ID:          "chatsen",
					Author:      "OrangeCat",
					Description: "Twitch chat client for iOS & Android with 7TV support",
					URL:         "https://chatsen.app",
				},
				{
					Name:        "DankChat",
					ID:          "dankchat",
					Author:      "flex3rs",
					Description: "Android Twitch Chat Client with 7TV support",
					URL:         "https://play.google.com/store/apps/details?id=com.flxrs.dankchat",
				},
			},
		},
	}

	return ctx.JSON(rest.OK, platforms)
}

type DownloadsResult struct {
	Platforms []Platform `json:"platforms"`
}

type Platform struct {
	ID         string            `mapstructure:"id" json:"id"`
	VersionTag string            `mapstructure:"version_tag" json:"version_tag"`
	New        bool              `mapstructure:"new" json:"new"`
	URL        string            `mapstructure:"url" json:"url,omitempty"`
	Variants   []PlatformVariant `mapstructure:"variants" json:"variants,omitempty"`
}

type PlatformVariant struct {
	Name        string `json:"name" mapstructure:"name"`
	ID          string `json:"id" mapstructure:"id"`
	Author      string `json:"author" mapstructure:"author"`
	Version     string `json:"version" mapstructure:"version"`
	Description string `json:"description" mapstructure:"description"`
	URL         string `json:"url" mapstructure:"url"`
}
