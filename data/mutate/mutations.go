package mutate

import (
	"sync"

	"github.com/seventv/common/events"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/svc"
	"github.com/seventv/common/svc/s3"
	"github.com/seventv/compactdisc"
)

type Mutate struct {
	id     svc.AppIdentity
	mongo  mongo.Instance
	redis  redis.Instance
	s3     s3.Instance
	events events.Instance
	cd     compactdisc.Instance
	mx     map[string]*sync.Mutex
}

func New(opt InstanceOptions) *Mutate {
	return &Mutate{
		id:     opt.ID,
		mongo:  opt.Mongo,
		redis:  opt.Redis,
		s3:     opt.S3,
		events: opt.Events,
		cd:     opt.CD,
		mx:     map[string]*sync.Mutex{},
	}
}

type InstanceOptions struct {
	ID     svc.AppIdentity
	Mongo  mongo.Instance
	Redis  redis.Instance
	S3     s3.Instance
	Events events.Instance
	CD     compactdisc.Instance
}
