package health

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/seventv/api/internal/configure"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/svc/s3"
	"github.com/seventv/api/internal/testutil"
	messagequeue "github.com/seventv/message-queue/go"
)

func TestHealth(t *testing.T) {
	config := &configure.Config{}
	config.Health.Enabled = true
	config.Health.Bind = "127.0.1.1:3000"

	gCtx, cancel := global.WithCancel(global.New(context.Background(), config))

	var err error
	gCtx.Inst().S3, err = s3.NewMock(gCtx, map[string]map[string][]byte{})
	testutil.IsNil(t, err, "s3 init successful")

	gCtx.Inst().MessageQueue, err = messagequeue.New(gCtx, messagequeue.ConfigMock{})
	testutil.IsNil(t, err, "mq init successful")

	mq, _ := gCtx.Inst().MessageQueue.(*messagequeue.InstanceMock)
	s3, _ := gCtx.Inst().S3.(*s3.MockInstance)

	// TODO we need to mock redis :-)
	// gCtx.Inst().Redis, err = redis.NewMock()
	// testutil.IsNil(t, err, "redis init successful")

	// TODO we need to mock mongo :-)
	// gCtx.Inst().Mongo, err = mongo.NewMock()
	// testutil.IsNil(t, err, "mongo init successful")

	done := New(gCtx)

	time.Sleep(time.Millisecond * 50)

	resp, err := http.DefaultClient.Get("http://127.0.1.1:3000")
	testutil.IsNil(t, err, "No error")

	_ = resp.Body.Close()
	testutil.Assert(t, http.StatusOK, resp.StatusCode, "response code all up")

	mq.SetConnected(false)

	resp, err = http.DefaultClient.Get("http://127.0.1.1:3000")
	testutil.IsNil(t, err, "No error")

	_ = resp.Body.Close()
	testutil.Assert(t, http.StatusInternalServerError, resp.StatusCode, "response code rmq down")

	mq.SetConnected(true)
	s3.SetConnected(false)

	resp, err = http.DefaultClient.Get("http://127.0.1.1:3000")
	testutil.IsNil(t, err, "No error")

	_ = resp.Body.Close()
	testutil.Assert(t, http.StatusInternalServerError, resp.StatusCode, "response code s3 down")

	cancel()

	<-done
}
