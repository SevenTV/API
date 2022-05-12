package v3

import (
	"context"
	goerrors "errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/SevenTV/Common/errors"
	"github.com/fasthttp/websocket"
	"go.uber.org/zap"

	wsTransport "github.com/seventv/api/internal/gql/websocket"

	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/gql/v3/cache"
	"github.com/seventv/api/internal/gql/v3/complexity"
	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/helpers"
	middlewarev3 "github.com/seventv/api/internal/gql/v3/middleware"
	"github.com/seventv/api/internal/gql/v3/resolvers"
	"github.com/seventv/api/internal/gql/v3/types"
	"github.com/seventv/api/internal/middleware"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func GqlHandlerV3(gCtx global.Context) func(ctx *fasthttp.RequestCtx) {
	schema := generated.NewExecutableSchema(generated.Config{
		Resolvers:  resolvers.New(types.Resolver{Ctx: gCtx}),
		Directives: middlewarev3.New(gCtx),
		Complexity: complexity.New(gCtx),
	})
	srv := handler.New(schema)
	exec := executor.New(schema)

	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.Use(extension.Introspection{})

	srv.Use(&extension.ComplexityLimit{
		Func: func(ctx context.Context, rc *graphql.OperationContext) int {
			return 75
		},
	})

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: cache.NewRedisCache(gCtx, "", time.Hour*6),
	})

	errorPresenter := func(ctx context.Context, e error) *gqlerror.Error {
		err := graphql.DefaultErrorPresenter(ctx, e)
		var apiErr errors.APIError
		if goerrors.As(e, &apiErr) {
			err.Message = fmt.Sprintf("%d %s", apiErr.Code(), apiErr.Message())
			err.Extensions = map[string]interface{}{
				"fields":  apiErr.GetFields(),
				"message": apiErr.Message(),
				"code":    apiErr.Code(),
			}
		}

		return err
	}
	srv.SetErrorPresenter(errorPresenter)
	exec.SetErrorPresenter(errorPresenter)

	srv.SetRecoverFunc(func(ctx context.Context, err interface{}) (userMessage error) {
		zap.S().Errorw("panic in gql handler",
			"panic", err,
		)
		return helpers.ErrInternalServerError
	})

	wsTransport := wsTransport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		InitFunc: func(ctx context.Context, initPayload wsTransport.InitPayload) (context.Context, error) {
			authHeader := initPayload.Authorization()

			if strings.HasPrefix(authHeader, "Bearer ") {
				tok := strings.TrimPrefix(authHeader, "Bearer ")

				user, err := middleware.DoAuth(gCtx, tok)
				if err != nil {
					goto handler
				}

				ctx = context.WithValue(ctx, helpers.UserKey, user)
			}

		handler:
			return ctx, nil
		},
		Upgrader: websocket.FastHTTPUpgrader{
			CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
				return true
			},
		},
	}

	return func(ctx *fasthttp.RequestCtx) {
		lCtx := context.WithValue(gCtx, helpers.UserKey, ctx.UserValue("user"))

		if wsTransport.Supports(ctx) {
			wsTransport.Do(ctx, lCtx, exec)
		} else {
			fasthttpadaptor.NewFastHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				srv.ServeHTTP(w, r.WithContext(lCtx))
			}))(ctx)
		}

	}
}
