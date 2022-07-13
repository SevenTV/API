package chatterino

import (
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/api/internal/rest/v3/middleware"
	"github.com/seventv/common/errors"
)

type Route struct {
	Ctx global.Context
}

func New(gCtx global.Context) rest.Route {
	return &Route{gCtx}
}

// Config implements rest.Route
func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/chatterino/version/{platform}/{branch}",
		Method: rest.GET,
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 1800, nil),
		},
	}
}

// Get Chatterino Update
// @Summary Auto update for chatterino
// @Description Allows chatterino clients to auto update
// @Tags chatterino
// @Param platform path string true "The platform such as win, linux, macos"
// @Param branch path string true "The branch such as stable or beta"
// @Produce json
// @Success 200 {object} VersionResult
// @Router /chatterino/version/{platform}/{branch} [get]
func (r *Route) Handler(ctx *rest.Ctx) errors.APIError {
	platform, _ := ctx.UserValue("platform").String()
	branch, _ := ctx.UserValue("branch").String()

	var (
		download         string
		portableDownload string
		updateExe        string
	)

	switch branch {
	case "stable":
		switch platform {
		case "win":
			download = r.Ctx.Config().Chatterino.Stable.Win.Download
			portableDownload = r.Ctx.Config().Chatterino.Stable.Win.PortableDownload
			updateExe = r.Ctx.Config().Chatterino.Stable.Win.UpdateExe
		case "linux":
			download = r.Ctx.Config().Chatterino.Stable.Linux.Download
			portableDownload = r.Ctx.Config().Chatterino.Stable.Linux.PortableDownload
			updateExe = r.Ctx.Config().Chatterino.Stable.Linux.UpdateExe
		case "macos":
			download = r.Ctx.Config().Chatterino.Stable.Macos.Download
			portableDownload = r.Ctx.Config().Chatterino.Stable.Macos.PortableDownload
			updateExe = r.Ctx.Config().Chatterino.Stable.Macos.UpdateExe
		}
	case "beta":
		switch platform {
		case "win":
			download = r.Ctx.Config().Chatterino.Beta.Win.Download
			portableDownload = r.Ctx.Config().Chatterino.Beta.Win.PortableDownload
			updateExe = r.Ctx.Config().Chatterino.Beta.Win.UpdateExe
		case "linux":
			download = r.Ctx.Config().Chatterino.Beta.Linux.Download
			portableDownload = r.Ctx.Config().Chatterino.Beta.Linux.PortableDownload
			updateExe = r.Ctx.Config().Chatterino.Beta.Linux.UpdateExe
		case "macos":
			download = r.Ctx.Config().Chatterino.Beta.Macos.Download
			portableDownload = r.Ctx.Config().Chatterino.Beta.Macos.PortableDownload
			updateExe = r.Ctx.Config().Chatterino.Beta.Macos.UpdateExe
		}
	}

	result := VersionResult{
		Download:         download,
		PortableDownload: portableDownload,
		UpdateExe:        updateExe,
		Version:          r.Ctx.Config().Chatterino.Version,
	}

	return ctx.JSON(rest.OK, result)
}

type VersionResult struct {
	Download         string `json:"download"`
	PortableDownload string `json:"portable_download,omitempty"`
	UpdateExe        string `json:"updateexe,omitempty"`
	Version          string `json:"version"`
}
