package pprof

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/seventv/api/internal/global"
	"go.uber.org/zap"
)

func New(gctx global.Context) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		if err := http.ListenAndServe(gctx.Config().PProf.Bind, nil); err != nil {
			zap.S().Fatalw("pprof failed to listen",
				"error", err,
			)
		}
	}()

	go func() {
		<-gctx.Done()
		close(done)
	}()

	return done
}
