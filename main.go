package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/redis"
	"github.com/SevenTV/Common/structures/v3/mutations"
	"github.com/SevenTV/Common/structures/v3/query"
	"github.com/bugsnag/panicwrap"
	"github.com/seventv/api/global"
	"github.com/seventv/api/global/configure"
	"github.com/seventv/api/gql"
	"github.com/sirupsen/logrus"
)

var (
	Version = "development"
	Unix    = ""
	Time    = "unknown"
	User    = "unknown"
)

func init() {
	debug.SetGCPercent(2000)
	if i, err := strconv.Atoi(Unix); err == nil {
		Time = time.Unix(int64(i), 0).Format(time.RFC3339)
	}
}

func main() {
	config := configure.New()

	exitStatus, err := panicwrap.BasicWrap(func(s string) {
		logrus.Error(s)
	})
	if err != nil {
		logrus.Error("failed to setup panic handler: ", err)
		os.Exit(2)
	}

	if exitStatus >= 0 {
		os.Exit(exitStatus)
	}

	if !config.NoHeader {
		logrus.Info("7TV API")
		logrus.Infof("Version: %s", Version)
		logrus.Infof("build.Time: %s", Time)
		logrus.Infof("build.User: %s", User)
	}

	logrus.Debug("MaxProcs: ", runtime.GOMAXPROCS(0))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	c, cancel := context.WithCancel(context.Background())

	gCtx := global.New(c, config)

	{
		// Set up Mongo
		ctx, cancel := context.WithTimeout(gCtx, time.Second*15)
		mongoInst, err := mongo.Setup(ctx, mongo.SetupOptions{
			URI: gCtx.Config().Mongo.URI,
			DB:  gCtx.Config().Mongo.DB,
		})
		cancel()
		if err != nil {
			logrus.WithError(err).Fatal("failed to connect to mongo")
		}

		ctx, cancel = context.WithTimeout(gCtx, time.Second*15)
		redisInst, err := redis.Setup(ctx, redis.SetupOptions{
			Username:  config.Redis.Username,
			Password:  config.Redis.Password,
			Database:  config.Redis.Database,
			Addresses: []string{gCtx.Config().Redis.URI},
		})
		cancel()
		if err != nil {
			logrus.WithError(err).Fatal("failed to connect to redis")
		}

		gCtx.Inst().Mongo = mongoInst
		gCtx.Inst().Redis = redisInst
		gCtx.Inst().Query = query.New(mongoInst, redisInst)
		gCtx.Inst().Mutate = mutations.New(mongoInst, redisInst)
	}

	serverDone := gql.New(gCtx)

	logrus.Info("running")

	done := make(chan struct{})
	go func() {
		<-sig
		cancel()
		go func() {
			select {
			case <-time.After(time.Minute):
			case <-sig:
			}
			logrus.Fatal("force shutdown")
		}()

		logrus.Info("shutting down")

		<-serverDone

		close(done)
	}()

	<-done

	logrus.Info("shutdown")
	os.Exit(0)

	cancel()
	if err != nil {
		logrus.WithError(err).Fatal("failed to connect to redis")
	}
}
