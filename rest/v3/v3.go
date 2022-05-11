package v3

import (
	"github.com/seventv/api/global"
	"github.com/seventv/api/rest/rest"
	"github.com/seventv/api/rest/v3/routes"
)

// @title 7TV REST API
// @version 3.0
// @description This is the REST API for 7TV
// @termsOfService TODO

// @contact.name 7TV Developers
// @contact.url https://discord.gg/7tv
// @contact.email dev@7tv.io

// @license.name Apache 2.0 + Commons Clause
// @license.url https://github.com/SevenTV/REST/blob/dev/LICENSE.md

// @host 7tv.io
// @BasePath /v3
// @schemes https
// @query.collection.format multi
func API(gCtx global.Context, router *rest.Router) rest.Route {
	return routes.New(gCtx)
}
