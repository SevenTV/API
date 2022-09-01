package portal

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasttemplate"
	"go.uber.org/zap"
)

func Serve(ctx context.Context) {
	wd, _ := os.Getwd()
	root := path.Join(wd, "portal", "dist")

	if root == "" {
		root = "/portal/dist"
	}

	// Setup FS handler
	fs := &fasthttp.FS{
		Root: root,
	}

	index, err := os.ReadFile(path.Join(root, "index.html"))
	if err != nil {
		zap.S().Warnw("couldn't begin serving dev portal", "error", err)

		return
	}

	favicon, err := os.ReadFile(path.Join(root, "ico.svg"))
	if err != nil {
		zap.S().Warnw("couldn't begin serving dev portal", "error", err)

		return
	}

	//replace-me
	template := fasttemplate.New(string(index), "<!-- {{", "}} -->")

	addr := os.Getenv("DEV_PORTAL_BIND")
	if addr == "" {
		addr = "0.0.0.0:3200"
	}

	// Start HTTP server.
	zap.S().Infow("Starting Portal Frontend", "addr", addr)

	go func() {
		handler := fs.NewRequestHandler()

		if err := fasthttp.ListenAndServe(addr, func(ctx *fasthttp.RequestCtx) {
			pth := string(ctx.Path())
			if strings.HasPrefix(pth, "/assets/") {
				handler(ctx)
			} else {
				if pth == "/ico.svg" {
					ctx.Response.Header.Set("Content-Type", "image/svg+xml")
					ctx.Response.Header.Set("Cache-Control", "max-age=3600")
					ctx.SetBody(favicon)
					return
				}

				ctx.Response.Header.Set("Content-Type", "text/html; charset=utf-8")
				ctx.Response.Header.Set("Cache-Control", "no-cache")
				ctx.SetBodyString(template.ExecuteString(map[string]interface{}{
					"META": "",
				}))
			}
		}); err != nil {
			log.Fatalf("error in ListenAndServe: %s", err)
		}
	}()

	log.Printf("Serving files from directory %s\n", root)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	// Wait forever.
	select {
	case <-ctx.Done():
	case <-sig:
	}
}
