package rest

import (
	"encoding/json"
	"runtime/debug"

	"github.com/fasthttp/router"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	v2 "github.com/seventv/api/internal/rest/v2"
	v3 "github.com/seventv/api/internal/rest/v3"
	"github.com/seventv/common/errors"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

func (s *HttpServer) V3(gCtx global.Context) {
	s.traverseRoutes(v3.API(gCtx, s.router), s.router)
}

func (s *HttpServer) V2(gCtx global.Context) {
	s.traverseRoutes(v2.API(gCtx, s.router), s.router)
}

func (s *HttpServer) SetupHandlers() {
	// Handle Not Found
	s.router.NotFound = s.getErrorHandler(
		rest.NotFound,
		errors.ErrUnknownRoute().SetFields(errors.Fields{
			"message": "The API endpoint requested does not exist",
		}),
	)

	// Handle P A N I C
	s.router.PanicHandler = func(ctx *fasthttp.RequestCtx, i interface{}) {
		err := "Uh oh. Something went horribly wrong"
		switch x := i.(type) {
		case error:
			err += ": " + x.Error()
		case string:
			err += ": " + x
		}
		zap.S().Errorw("panic occured",
			"panic", i,
			"stack", debug.Stack(),
		)
		s.getErrorHandler(
			rest.InternalServerError,
			errors.ErrInternalServerError().SetFields(errors.Fields{
				"panic": err,
			}),
		)(ctx)
	}
}

func (s *HttpServer) traverseRoutes(r rest.Route, parentGroup Router) {
	c := r.Config()

	// Compose the full request URI (prefixing with parent, if any)
	routable := parentGroup
	group := routable.Group(c.URI)
	l := zap.S().With(
		"group", group,
		"method", c.Method,
	)

	// Handle requests
	group.Handle(string(c.Method), "", func(ctx *fasthttp.RequestCtx) {
		rctx := &rest.Ctx{RequestCtx: ctx}

		handlers := make([]rest.Middleware, len(c.Middleware)+1)
		copy(handlers, c.Middleware)
		handlers[len(handlers)-1] = r.Handler

		for _, h := range handlers {
			if err := h(rctx); err != nil {
				// If the request handler returned an error
				// we will format it into standard API error response
				if ctx.Response.StatusCode() < 400 {
					rctx.SetStatusCode(rest.HttpStatusCode(err.ExpectedHTTPStatus()))
				}
				resp := &rest.APIErrorResponse{
					Status:     rctx.StatusCode().String(),
					StatusCode: rctx.StatusCode(),
					Error:      err.Message(),
					ErrorCode:  err.Code(),
					Details:    err.GetFields(),
				}

				b, _ := json.Marshal(resp)
				rctx.SetContentType("application/json")
				rctx.SetBody(b)
				return
			}
		}
	})
	l.Debug("Route registered")

	// activate child routes
	for _, child := range c.Children {
		s.traverseRoutes(child, group)
	}
}

func (s *HttpServer) getErrorHandler(status rest.HttpStatusCode, err rest.APIError) func(ctx *fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		b, _ := json.Marshal(&rest.APIErrorResponse{
			Status:     status.String(),
			StatusCode: status,
			Error:      err.Message(),
			ErrorCode:  err.Code(),
			Details:    err.GetFields(),
		})
		ctx.SetContentType("application/json")
		ctx.SetBody(b)
	}
}

type Router interface {
	ANY(path string, handler fasthttp.RequestHandler)
	CONNECT(path string, handler fasthttp.RequestHandler)
	DELETE(path string, handler fasthttp.RequestHandler)
	GET(path string, handler fasthttp.RequestHandler)
	Group(path string) *router.Group
	HEAD(path string, handler fasthttp.RequestHandler)
	Handle(method, path string, handler fasthttp.RequestHandler)
	OPTIONS(path string, handler fasthttp.RequestHandler)
	PATCH(path string, handler fasthttp.RequestHandler)
	POST(path string, handler fasthttp.RequestHandler)
	PUT(path string, handler fasthttp.RequestHandler)
	ServeFiles(path string, rootPath string)
	ServeFilesCustom(path string, fs *fasthttp.FS)
	TRACE(path string, handler fasthttp.RequestHandler)
}
