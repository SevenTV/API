package v2

import (
	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/api/rest/v2/routes"
	"github.com/seventv/api/internal/global"
)

// @title 7TV REST API
// @version 2.0
// @description This is the former v2 REST API for 7TV (deprecated)
// @termsOfService TODO

// @contact.name 7TV Developers
// @contact.url https://discord.gg/7tv
// @contact.email dev@7tv.io

// @license.name Apache 2.0 + Commons Clause
// @license.url https://github.com/SevenTV/REST/blob/dev/LICENSE.md

// @host api.7tv.app
// @BasePath /v2
// @schemes https
// @query.collection.format multi
func API(gCtx global.Context, router *rest.Router) rest.Route {
	return routes.New(gCtx)
}
