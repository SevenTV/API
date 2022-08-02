package query

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"time"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func (r *Resolver) emoteCategoryTrending(ctx context.Context, opt trendingCategoryOptions) ([]primitive.ObjectID, map[primitive.ObjectID]uint64, error) {
	// Lock this globally
	// this method should never, ever run more than once at a time. It's very expensive!
	mxKey := r.Ctx.Inst().Redis.ComposeKey("api-gql", "lock", "trending-emotes")
	mx := r.Ctx.Inst().Redis.Mutex(mxKey, time.Second)

	if err := mx.Lock(); err != nil {
		zap.S().Errorw("Failed to acquire lock for trending emotes", "error", err)

		return nil, nil, errors.ErrInternalServerError()
	}

	defer func() {
		if _, err := mx.Unlock(); err != nil {
			zap.S().Errorw("Failed to release lock for trending emotes", "error", err)
		}
	}()

	h := sha256.New()
	h.Write(utils.S2B(strconv.Itoa(int(opt.Days))))
	h.Write(utils.S2B(strconv.Itoa(int(opt.EmoteMaxAge))))
	h.Write(utils.S2B(strconv.Itoa(int(opt.UsageThresold))))
	h.Write(utils.S2B(strconv.Itoa(int(opt.UserMinAge))))

	// Retrieve current cache
	cacheKey := r.Ctx.Inst().Redis.ComposeKey("api-gql", "trending-emotes", hex.EncodeToString((h.Sum(nil))))
	cacheResult1, _ := r.Ctx.Inst().Redis.Get(ctx, cacheKey+":ids")
	cacheResult2, _ := r.Ctx.Inst().Redis.Get(ctx, cacheKey+":countmap")

	if cacheResult1 != "" && cacheResult2 != "" {
		// Unmarshal the cache
		ids := []primitive.ObjectID{}
		if err := json.Unmarshal(utils.S2B(cacheResult1), &ids); err != nil {
			zap.S().Errorw("Failed to unmarshal cache 1 for trending emotes", "error", err)

			return nil, nil, errors.ErrInternalServerError()
		}

		m := map[primitive.ObjectID]uint64{}
		if err := json.Unmarshal(utils.S2B(cacheResult2), &m); err != nil {
			zap.S().Errorw("Failed to unmarshal cache 2 for trending emotes", "error", err)

			return nil, nil, errors.ErrInternalServerError()
		}

		// Return cache
		return ids, m, nil
	}

	// The date at which to start gauging trending activity
	maxDate := time.Now()
	if opt.TimeTravel != nil {
		maxDate = *opt.TimeTravel
	}

	minDate := maxDate.Add(-time.Duration(opt.Days) * 24 * time.Hour)

	// Retrieve additions
	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameAuditLogs).Aggregate(ctx, mongo.Pipeline{
		{{
			Key: "$match",
			Value: bson.M{
				"_id": bson.M{
					"$gte": primitive.NewObjectIDFromTimestamp(minDate),
					"$lte": primitive.NewObjectIDFromTimestamp(maxDate),
				},
				"kind":        structures.AuditLogKindUpdateEmoteSet,
				"target_kind": structures.ObjectKindEmoteSet,
				"changes.key": "emotes",
			},
		}},
		{{
			Key: "$project",
			Value: bson.M{
				"target_id": "$target_id",
				"actor_id":  "$actor_id",
				"emote_ids": bson.M{"$first": "$changes.value.added.id"},
			},
		}},
	})
	if err != nil {
		zap.S().Errorw("mongo, failed to create query for activity data for trending emotes", "error", err)

		return nil, nil, errors.ErrInternalServerError()
	}

	seenMap := make(map[primitive.ObjectID]utils.Set[[24]byte])

	useMap := make(map[primitive.ObjectID]uint64)

	for cur.Next(ctx) {
		doc := aggregatedEmoteTrendingActivity{}

		if err := cur.Decode(&doc); err != nil {
			zap.S().Errorw("mongo, failed to decode activity data for trending emotes")

			return nil, nil, errors.ErrInternalServerError()
		}

		// Check that actor's account is old enough
		actorAge := time.Since(doc.ActorID.Timestamp()).Hours() / 24
		if actorAge < float64(opt.UserMinAge) {
			continue // skip because actor's account is too new to count
		}

		aID := [12]byte{} // actor id
		copy(aID[:], doc.ActorID[:])

		tID := [12]byte{} // target id
		copy(tID[:], doc.TargetID[:])

		// Merge the target + actor id
		// we will use this to filter out emote re-adds by one user
		mergedID := [24]byte{}
		for i, b := range append(aID[:], tID[:]...) {
			mergedID[i] = b
		}

		for _, emoteID := range doc.EmoteIDs {
			emoteAge := time.Since(emoteID.Timestamp()).Hours() / 24
			if opt.EmoteMaxAge > 0 && emoteAge > float64(opt.EmoteMaxAge) {
				continue // skip if the emote is too old to count
			}

			// Create or get seen map
			sm, ok := seenMap[emoteID]
			if !ok {
				sm = utils.Set[[24]byte]{}
				seenMap[emoteID] = sm
			}
			// Actor/Set has already been seen for this emote
			if sm.Has(mergedID) {
				continue // skip
			}

			// Increment use map for the emote
			useMap[emoteID]++

			sm.Add(mergedID)
		}
	}

	result := []primitive.ObjectID{}

	for emoteID, count := range useMap {
		if count < opt.UsageThresold {
			continue // skip if below the usage thresold
		}

		result = append(result, emoteID)
	}

	sort.Slice(result, func(i, j int) bool {
		return useMap[result[i]] > useMap[result[j]]
	})

	// Slice off results beyond 100
	if opt.Limit > 0 && len(result) > opt.Limit {
		result = result[:opt.Limit]
	}

	// Store results to cache
	j1, _ := json.Marshal(result)
	_ = r.Ctx.Inst().Redis.SetEX(ctx, cacheKey+":ids", j1, time.Minute*5)

	j2, _ := json.Marshal(useMap)
	_ = r.Ctx.Inst().Redis.SetEX(ctx, cacheKey+":countmap", j2, time.Minute*5)

	return result, useMap, nil
}

type trendingCategoryOptions struct {
	Days          uint32
	UserMinAge    uint32
	EmoteMaxAge   uint32
	UsageThresold uint64
	Limit         int

	TimeTravel *time.Time
}

type aggregatedEmoteTrendingActivity struct {
	TargetID primitive.ObjectID   `bson:"target_id"`
	ActorID  primitive.ObjectID   `bson:"actor_id"`
	EmoteIDs []primitive.ObjectID `bson:"emote_ids"`
}
