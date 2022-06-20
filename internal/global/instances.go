package global

import (
	"github.com/seventv/api/internal/instance"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/structures/v3/mutations"
	"github.com/seventv/common/structures/v3/query"
)

type Instances struct {
	Mongo      mongo.Instance
	Redis      redis.Instance
	S3         instance.S3
	RMQ        instance.RMQ
	Prometheus instance.Prometheus
	Loaders    instance.Loaders

	Query  *query.Query
	Mutate *mutations.Mutate
}
