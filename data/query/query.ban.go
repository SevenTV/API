package query

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (q *Query) Bans(ctx context.Context, opt BanQueryOptions) (*BanQueryResult, error) {
	mtx := q.mtx("bans")
	mtx.Lock()
	defer mtx.Unlock()

	filter := bson.M{}
	for k, v := range opt.Filter {
		filter[k] = v
	}

	// Define cache key
	hs := "all"
	if len(filter) > 0 {
		f, _ := json.Marshal(filter)
		h := sha256.New()
		h.Write(f)
		hs = hex.EncodeToString(h.Sum(nil))
	}
	k := q.key(fmt.Sprintf("bans:%s", hs))
	filter["expire_at"] = bson.M{"$gt": time.Now()}

	r := &BanQueryResult{
		All:           []structures.Ban{},
		NoPermissions: BanMap{},
		NoAuth:        BanMap{},
		NoOwnership:   BanMap{},
		MemoryHole:    BanMap{},
	}
	bans := []*aggregatedBansResult{}
	formatResult := func() *BanQueryResult {
		for _, g := range bans {
			victimID := g.UserID
			for _, ban := range g.Bans {
				r.All = append(r.All, ban)

				if ban.Effects.Has(structures.BanEffectNoPermissions) {
					r.NoPermissions[victimID] = ban
				}
				if ban.Effects.Has(structures.BanEffectNoAuth) {
					r.NoAuth[victimID] = ban
				}
				if ban.Effects.Has(structures.BanEffectNoOwnership) {
					r.NoOwnership[victimID] = ban
				}
				if ban.Effects.Has(structures.BanEffectMemoryHole) {
					r.MemoryHole[victimID] = ban
				}
			}
		}

		return r
	}

	// Get cached items
	if ok := q.getFromMemCache(ctx, k, &bans); ok {
		return formatResult(), nil
	}

	// Query
	cur, err := q.mongo.Collection(mongo.CollectionNameBans).Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{
			Key: "$group",
			Value: bson.M{
				"_id": "$victim_id",
				"bans": bson.M{
					"$push": "$$ROOT",
				},
			},
		}},
	})
	if err == nil {
		if err = cur.All(ctx, &bans); err != nil {
			return r, err
		}
	}

	// Set cache
	if err = q.setInMemCache(ctx, k, &bans, time.Second*1); err != nil {
		return r, err
	}
	return formatResult(), nil
}

type BanQueryOptions struct {
	Filter bson.M
}

type aggregatedBansResult struct {
	UserID primitive.ObjectID `json:"id" bson:"_id"`
	Bans   []structures.Ban   `json:"bans" bson:"bans"`
}

type BanQueryResult struct {
	All []structures.Ban
	// A list of user IDs which will not have any permissions at all
	NoPermissions BanMap
	// A list of user IDs not allowed to authenticate
	NoAuth BanMap
	// A list of user IDs who own nothing and are happy
	NoOwnership BanMap
	// A list of user IDs in the memory hole
	// (filtered from API results)
	MemoryHole BanMap
}

// BanMap is a map of user IDs to a ban object
type BanMap map[primitive.ObjectID]structures.Ban

// KeySlice returns a slice of user IDs for the ban map
func (bm BanMap) KeySlice() []primitive.ObjectID {
	v := make([]primitive.ObjectID, len(bm))
	ind := 0
	for k := range bm {
		v[ind] = k
		ind++
	}
	return v
}

func (bm BanMap) Get(id primitive.ObjectID) structures.Ban {
	return bm[id]
}

func (bm BanMap) Has(id primitive.ObjectID) bool {
	_, ok := bm[id]

	return ok
}
