package global

import (
	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/redis"
	"github.com/SevenTV/Common/structures/v3/mutations"
	"github.com/SevenTV/Common/structures/v3/query"
)

type Instances struct {
	Mongo  mongo.Instance
	Redis  redis.Instance
	Query  *query.Query
	Mutate *mutations.Mutate
}
