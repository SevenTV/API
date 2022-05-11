package global

import (
	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/redis"
	"github.com/SevenTV/Common/structures/v3/mutations"
	"github.com/SevenTV/Common/structures/v3/query"
	"github.com/seventv/api/global/instance"
)

type Instances struct {
	Mongo  mongo.Instance
	Redis  redis.Instance
	AwsS3  instance.AwsS3
	Rmq    instance.Rmq
	Query  *query.Query
	Mutate *mutations.Mutate
}
