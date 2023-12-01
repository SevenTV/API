package instance

import (
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/svc/s3"
	"github.com/seventv/compactdisc"
	messagequeue "github.com/seventv/message-queue/go"

	"github.com/seventv/api/data/events"
	"github.com/seventv/api/data/model"
	"github.com/seventv/api/data/mutate"
	"github.com/seventv/api/data/query"
	"github.com/seventv/api/internal/loaders"
	"github.com/seventv/api/internal/search"
	"github.com/seventv/api/internal/svc/auth"
	"github.com/seventv/api/internal/svc/limiter"
	"github.com/seventv/api/internal/svc/presences"
	"github.com/seventv/api/internal/svc/prometheus"
	"github.com/seventv/api/internal/svc/youtube"
)

type Instances struct {
	Mongo        mongo.Instance
	Meilisearch  *search.MeiliSearch
	Redis        redis.Instance
	Auth         auth.Authorizer
	S3           s3.Instance
	MessageQueue messagequeue.Instance
	Prometheus   prometheus.Instance
	Events       events.Instance
	Limiter      limiter.Instance
	YouTube      youtube.Instance
	Loaders      loaders.Instance
	Presences    presences.Instance
	Modelizer    model.Modelizer
	CD           compactdisc.Instance

	Query  *query.Query
	Mutate *mutate.Mutate
}
