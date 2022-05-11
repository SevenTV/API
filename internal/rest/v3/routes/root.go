package routes

import (
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/api/internal/rest/v3/middleware"
	"github.com/seventv/api/internal/rest/v3/routes/auth"
	"github.com/seventv/api/internal/rest/v3/routes/docs"
	"github.com/seventv/api/internal/rest/v3/routes/emotes"
)

type Route struct {
	Ctx global.Context
}

func New(gCtx global.Context) rest.Route {
	return &Route{gCtx}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/v3",
		Method: rest.GET,
		Children: []rest.Route{
			docs.New(r.Ctx),
			auth.New(r.Ctx),
			emotes.New(r.Ctx),
		},
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 30, nil),
		},
	}
}

func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	// uptime := r.Ctx.Value("uptime").(time.Time)

	// // Default service statuses
	// services := responseServices{
	// 	RabbitMQ: responseServiceStatusUnavailable,
	// 	S3:       responseServiceStatusUnavailable,
	// 	MongoDB:  responseServiceStatusUnavailable,
	// 	Redis:    responseServiceStatusUnavailable,
	// }

	// // Define a context that will last one second
	// lctx, cancel := context.WithTimeout(ctx, time.Second*1)
	// wg := sync.WaitGroup{}
	// if r.Ctx.Inst().Mongo != nil {
	// 	wg.Add(1)
	// 	go func() {
	// 		defer wg.Done()

	// 		if err := r.Ctx.Inst().Mongo.Ping(lctx); err == nil {
	// 			services.MongoDB = responseServiceStatusOK
	// 		} else if lctx.Err() != nil {
	// 			services.MongoDB = responseServiceStatusTimeout
	// 		}
	// 	}()
	// }
	// if r.Ctx.Inst().Redis != nil {
	// 	wg.Add(1)
	// 	go func() {
	// 		defer wg.Done()

	// 		if err := r.Ctx.Inst().Redis.Ping(lctx); err == nil {
	// 			services.Redis = responseServiceStatusOK
	// 		} else if lctx.Err() != nil {
	// 			services.Redis = responseServiceStatusTimeout
	// 		}
	// 	}()
	// }
	// wg.Wait()
	// cancel()

	return ctx.JSON(rest.OK, &Response{
		Online: true,
		// Uptime:   uptime.Format(time.RFC3339),
		// Services: services,
	})
}

type Response struct {
	Online bool `json:"online"`
	// Uptime   string           `json:"uptime"`
	// Services responseServices `json:"services"`
}

// type responseServices struct {
// 	MongoDB  responseServiceStatus `json:"database"`
// 	Redis    responseServiceStatus `json:"memcache"`
// 	RabbitMQ responseServiceStatus `json:"tasks"`
// 	S3       responseServiceStatus `json:"objectstore"`
// }

// type responseServiceStatus string

// const (
// 	responseServiceStatusOK          responseServiceStatus = "OK"
// 	responseServiceStatusUnavailable responseServiceStatus = "UNAVAILABLE"
// 	responseServiceStatusTimeout     responseServiceStatus = "TIMED_OUT"
// )
