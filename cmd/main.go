package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/bugsnag/panicwrap"
	"github.com/nats-io/nats.go"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/mongo/indexing"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/svc"
	"github.com/seventv/common/svc/s3"
	"github.com/seventv/compactdisc"
	messagequeue "github.com/seventv/message-queue/go"
	"go.uber.org/zap"

	"github.com/seventv/api/data/events"
	"github.com/seventv/api/data/model"
	"github.com/seventv/api/data/mutate"
	"github.com/seventv/api/data/query"
	"github.com/seventv/api/internal/api/eventbridge"
	"github.com/seventv/api/internal/api/gql"
	"github.com/seventv/api/internal/api/rest"
	"github.com/seventv/api/internal/configure"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/loaders"
	"github.com/seventv/api/internal/search"
	"github.com/seventv/api/internal/svc/auth"
	"github.com/seventv/api/internal/svc/health"
	"github.com/seventv/api/internal/svc/limiter"
	"github.com/seventv/api/internal/svc/monitoring"
	"github.com/seventv/api/internal/svc/pprof"
	"github.com/seventv/api/internal/svc/presences"
	"github.com/seventv/api/internal/svc/prometheus"
	"github.com/seventv/api/internal/svc/youtube"
)

var (
	Version = "development"
	Unix    = ""
	Time    = "unknown"
	User    = "unknown"
)

func init() {
	if i, err := strconv.Atoi(Unix); err == nil {
		Time = time.Unix(int64(i), 0).Format(time.RFC3339)
	}
}

func main() {
	config := configure.New()

	exitStatus, err := panicwrap.BasicWrap(func(s string) {
		zap.S().Errorw("panic detected",
			"panic", s,
		)
	})
	if err != nil {
		zap.S().Errorw("failed to setup panic handler",
			"error", err,
		)
		os.Exit(2)
	}

	if exitStatus >= 0 {
		os.Exit(exitStatus)
	}

	if !config.NoHeader {
		zap.S().Info("7TV API")
		zap.S().Infof("Version: %s", Version)
		zap.S().Infof("build.Time: %s", Time)
		zap.S().Infof("build.User: %s", User)
	}

	zap.S().Debugf("MaxProcs: %d", runtime.GOMAXPROCS(0))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	gctx, cancel := global.WithCancel(global.New(context.Background(), config))

	{
		gctx.Inst().Redis, err = redis.Setup(gctx, redis.SetupOptions{
			Username:   config.Redis.Username,
			Password:   config.Redis.Password,
			Database:   config.Redis.Database,
			Sentinel:   config.Redis.Sentinel,
			Addresses:  config.Redis.Addresses,
			MasterName: config.Redis.MasterName,
			EnableSync: true,
		})
		if err != nil {
			zap.S().Fatalw("failed to setup redis handler",
				"error", err,
			)
		}
	}

	// INITIALIZE MEILISEARCH
	gctx.Inst().Meilisearch = search.New(gctx)

	{
		gctx.Inst().Mongo, err = mongo.Setup(gctx, mongo.SetupOptions{
			URI:      config.Mongo.URI,
			DB:       config.Mongo.DB,
			Direct:   config.Mongo.Direct,
			Username: config.Mongo.Username,
			Password: config.Mongo.Password,
		})
		if err != nil {
			zap.S().Fatalw("failed to setup mongo handler",
				"error", err,
			)
		}

		// Run collsync
		go func() {
			if err := indexing.CollSync(gctx.Inst().Mongo, indexing.DatabaseRefAPI); err != nil {
				zap.S().Errorw("couldn't set up indexes",
					"error", err,
				)
			}
		}()
	}

	{
		switch config.MessageQueue.Mode {
		case configure.MessageQueueModeRMQ:
			gctx.Inst().MessageQueue, err = messagequeue.New(gctx, messagequeue.ConfigRMQ{
				AmqpURI:              config.MessageQueue.RMQ.URI,
				MaxReconnectAttempts: config.MessageQueue.RMQ.MaxReconnectAttempts,
			})
		case configure.MessageQueueModeSQS:
			gctx.Inst().MessageQueue, err = messagequeue.New(gctx, messagequeue.ConfigSQS{
				Region: config.MessageQueue.SQS.Region,
				Credentials: aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
					return aws.Credentials{
						AccessKeyID:     config.MessageQueue.SQS.AccessToken,
						SecretAccessKey: config.MessageQueue.SQS.SecretKey,
					}, nil
				}),
				RetryMaxAttempts: config.MessageQueue.SQS.MaxRetryAttempts,
			})
		}
		if err != nil {
			zap.S().Fatalw("failed to setup mq handler",
				"error", err,
			)
		}
	}

	{
		gctx.Inst().Auth = auth.New(gctx, auth.AuthorizerOptions{
			JWTSecret: config.Credentials.JWTSecret,
			Domain:    config.Http.Cookie.Domain,
			Secure:    config.Http.Cookie.Secure,
			Config:    config.Platforms,
			Redis:     gctx.Inst().Redis,
		})
	}

	{
		gctx.Inst().S3, err = s3.New(gctx, s3.Options{
			Region:      config.S3.Region,
			Endpoint:    config.S3.Endpoint,
			AccessToken: config.S3.AccessToken,
			SecretKey:   config.S3.SecretKey,
			Namespace:   config.S3.Namespace,
		})
		if err != nil {
			zap.S().Warnw("failed to setup s3 handler",
				"error", err,
			)
		}
	}

	{
		gctx.Inst().Prometheus = prometheus.New(prometheus.Options{
			Labels: config.Monitoring.Labels.ToPrometheus(),
		})
	}

	nc, err := nats.Connect(config.Nats.Url)
	if err != nil {
		zap.S().Fatalw("failed to connect to nats",
			"error", err,
		)
	}

	defer func() {
		err = nc.Drain()
		zap.S().Fatalw("failed to drain nats, is connection failing?",
			"error", err,
		)
	}()

	gctx.Inst().Events = events.NewPublisher(nc, config.Nats.Subject)

	{
		id := svc.AppIdentity{
			Name: "API",
			Web:  config.WebsiteURL,
			CDN:  config.CdnURL,
		}

		gctx.Inst().Limiter, err = limiter.New(gctx, gctx.Inst().Redis)
		if err != nil {
			zap.S().Fatalw("failed to setup rate limiter", "error", err)
		}

		gctx.Inst().CD = compactdisc.New(config.Platforms.Discord.API)

		gctx.Inst().Modelizer = model.NewInstance(model.ModelInstanceOptions{
			CDN:     config.CdnURL,
			Website: config.WebsiteURL,
		})
		gctx.Inst().Query = query.New(gctx.Inst().Mongo, gctx.Inst().Redis, gctx.Inst().Meilisearch)
		gctx.Inst().Loaders = loaders.New(gctx, gctx.Inst().Mongo, gctx.Inst().Redis, gctx.Inst().Query)

		gctx.Inst().Mutate = mutate.New(mutate.InstanceOptions{
			ID:        id,
			Mongo:     gctx.Inst().Mongo,
			Loaders:   gctx.Inst().Loaders,
			Redis:     gctx.Inst().Redis,
			S3:        gctx.Inst().S3,
			Modelizer: gctx.Inst().Modelizer,
			Events:    gctx.Inst().Events,
			CD:        gctx.Inst().CD,
		})

		gctx.Inst().Presences = presences.New(presences.Options{
			Mongo:     gctx.Inst().Mongo,
			Loaders:   gctx.Inst().Loaders,
			Events:    gctx.Inst().Events,
			Config:    config,
			Modelizer: gctx.Inst().Modelizer,
		})
	}

	{
		gctx.Inst().YouTube, err = youtube.New(gctx, youtube.YouTubeOptions{
			APIKey: config.Platforms.YouTube.APIKey,
		})
		if err != nil {
			zap.S().Errorw("failed to setup youtube instance",
				"error", err,
			)
		}
	}

	wg := sync.WaitGroup{}

	if gctx.Config().Health.Enabled {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-health.New(gctx)
		}()
	}

	if gctx.Config().Monitoring.Enabled {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-monitoring.New(gctx)
		}()
	}

	if gctx.Config().PProf.Enabled {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-pprof.New(gctx)
		}()
	}

	if gctx.Config().EventBridge.Enabled {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-eventbridge.New(gctx)
		}()
	}

	done := make(chan struct{})

	go func() {
		<-sig
		cancel()

		go func() {
			select {
			case <-time.After(time.Minute):
			case <-sig:
			}
			zap.S().Fatal("force shutdown")
		}()

		zap.S().Info("shutting down")

		wg.Wait()

		close(done)
	}()

	wg.Add(2)

	go func() {
		defer wg.Done()

		if err := rest.New(gctx); err != nil {
			zap.S().Fatalw("rest failed",
				"error", err,
			)
		}
	}()

	go func() {
		defer wg.Done()

		if err := gql.New(gctx); err != nil {
			zap.S().Fatalw("gql failed",
				"error", err,
			)
		}
	}()

	zap.S().Info("running")

	<-done

	zap.S().Info("shutdown")
	os.Exit(0)
}
