package v2

import (
	"context"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/gql/v2/complexity"
	"github.com/seventv/api/internal/gql/v2/gen/generated"
	"github.com/seventv/api/internal/gql/v2/helpers"
	gql_middleware "github.com/seventv/api/internal/gql/v2/middleware"
	"github.com/seventv/api/internal/gql/v2/resolvers"
	"github.com/seventv/api/internal/gql/v2/types"
	"github.com/seventv/api/internal/gql/v3/cache"
	"github.com/seventv/api/internal/middleware"
	"github.com/seventv/common/errors"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.uber.org/zap"
)

func GqlHandlerV2(gCtx global.Context) func(ctx *fasthttp.RequestCtx) {
	schema := generated.NewExecutableSchema(generated.Config{
		Resolvers:  resolvers.New(types.Resolver{Ctx: gCtx}),
		Directives: gql_middleware.New(gCtx),
		Complexity: complexity.New(gCtx),
	})
	srv := handler.New(schema)

	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.Use(extension.Introspection{})

	srv.Use(&extension.ComplexityLimit{
		Func: func(ctx context.Context, rc *graphql.OperationContext) int {
			return 100
		},
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

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: cache.NewRedisCache(gCtx, "", time.Hour*6),
	})

	srv.SetRecoverFunc(func(ctx context.Context, err interface{}) (userMessage error) {
		zap.S().Error("panic in gql handler",
			"panic", err,
		)
		return errors.ErrInternalServerError()
	})

	rateLimitFunc := middleware.RateLimit(gCtx, "gql-v3", gCtx.Config().Limits.Buckets.GQL2[0], time.Second*time.Duration(gCtx.Config().Limits.Buckets.GQL2[1]))

	checkLimit := func(ctx *fasthttp.RequestCtx) bool {
		if err := rateLimitFunc(ctx); err != nil {
			j, _ := json.Marshal(errorPresenter(ctx, err))

			ctx.SetContentType("application/json")
			ctx.SetBody(j)

			return false
		}
		return true
	}

	return func(ctx *fasthttp.RequestCtx) {
		lCtx := context.WithValue(gCtx, helpers.UserKey, ctx.UserValue("user"))
		lCtx = context.WithValue(lCtx, helpers.RequestCtxKey, ctx)

		if ok := checkLimit(ctx); !ok {
			return
		}

		fasthttpadaptor.NewFastHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			srv.ServeHTTP(w, r.WithContext(lCtx))
		}))(ctx)
	}
}
