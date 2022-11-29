package mutate

import (
	"sync"

	"github.com/seventv/api/data/events"
	"github.com/seventv/api/data/model"
	"github.com/seventv/api/internal/loaders"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/svc"
	"github.com/seventv/common/svc/s3"
	"github.com/seventv/compactdisc"
)

type Mutate struct {
	id        svc.AppIdentity
	mongo     mongo.Instance
	loaders   loaders.Instance
	redis     redis.Instance
	s3        s3.Instance
	modelizer model.Modelizer
	events    events.Instance
	cd        compactdisc.Instance
	mx        map[string]*sync.Mutex
}

func New(opt InstanceOptions) *Mutate {
	return &Mutate{
		id:        opt.ID,
		mongo:     opt.Mongo,
		loaders:   opt.Loaders,
		redis:     opt.Redis,
		s3:        opt.S3,
		modelizer: opt.Modelizer,
		events:    opt.Events,
		cd:        opt.CD,
		mx:        map[string]*sync.Mutex{},
	}
}

type InstanceOptions struct {
	ID        svc.AppIdentity
	Mongo     mongo.Instance
	Loaders   loaders.Instance
	Redis     redis.Instance
	S3        s3.Instance
	Modelizer model.Modelizer
	Events    events.Instance
	CD        compactdisc.Instance
}
