package global

import (
	"github.com/seventv/api/internal/instance"
	"github.com/seventv/api/internal/svc/prometheus"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/structures/v3/mutations"
	"github.com/seventv/common/structures/v3/query"
	"github.com/seventv/common/svc/s3"
	messagequeue "github.com/seventv/message-queue/go"
)

type Instances struct {
	Mongo        mongo.Instance
	Redis        redis.Instance
	S3           s3.Instance
	MessageQueue messagequeue.Instance
	Prometheus   prometheus.Instance
	Loaders      instance.Loaders

	Query  *query.Query
	Mutate *mutations.Mutate
}
