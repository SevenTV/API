package global

import (
	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/redis"
	"github.com/SevenTV/Common/structures/v3/mutations"
	"github.com/SevenTV/Common/structures/v3/query"
	"github.com/meilisearch/meilisearch-go"
	"github.com/seventv/api/internal/instance"
)

type Instances struct {
	Mongo        mongo.Instance
	Redis        redis.Instance
	S3           instance.S3
	RMQ          instance.RMQ
	Prometheus   instance.Prometheus
	Loaders      instance.Loaders
	MeilieSearch *meilisearch.Client

	Query  *query.Query
	Mutate *mutations.Mutate
}
