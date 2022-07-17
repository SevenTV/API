package user

import (
	"context"
	"sort"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/api/internal/gql/v3/types"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.UserResolver {
	return &Resolver{r}
}

func (r *Resolver) EmoteSets(ctx context.Context, obj *model.User) ([]*model.EmoteSet, error) {
	sets, err := r.Ctx.Inst().Loaders.EmoteSetByUserID().Load(obj.ID)
	if err != nil {
		return nil, err
	}

	result := make([]*model.EmoteSet, len(sets))
	for i, v := range sets {
		result[i] = helpers.EmoteSetStructureToModel(v, r.Ctx.Config().CdnURL)
	}

	return result, nil
}

// Connections lists the users' connections
func (r *Resolver) Connections(ctx context.Context, obj *model.User, platforms []model.ConnectionPlatform) ([]*model.UserConnection, error) {
	result := []*model.UserConnection{}

	for _, conn := range obj.Connections {
		ok := false

		if len(platforms) > 0 {
			for _, p := range platforms {
				if conn.Platform == p {
					ok = true
					break
				}
			}
		} else {
			ok = true
		}

		if ok {
			result = append(result, conn)
		}
	}

	return result, nil
}

// Editors returns a users' list of editors
func (r *Resolver) Editors(ctx context.Context, obj *model.User) ([]*model.UserEditor, error) {
	ids := make([]primitive.ObjectID, len(obj.Editors))
	for i, v := range obj.Editors {
		ids[i] = v.ID
	}

	users, errs := r.Ctx.Inst().Loaders.UserByID().LoadAll(ids)
	result := []*model.UserEditor{}

	for _, e := range obj.Editors {
		for _, u := range users {
			if e.ID == u.ID {
				e.User = helpers.UserStructureToPartialModel(helpers.UserStructureToModel(u, r.Ctx.Config().CdnURL))
				result = append(result, e)

				break
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		a := result[i]
		b := result[j]

		return a.AddedAt.After(b.AddedAt)
	})

	return result, multierror.Append(nil, errs...).ErrorOrNil()
}

func (r *Resolver) EditorOf(ctx context.Context, obj *model.User) ([]*model.UserEditor, error) {
	result := []*model.UserEditor{}

	editables, err := r.Ctx.Inst().Query.UserEditorOf(ctx, obj.ID)
	if err == nil {
		for _, ed := range editables {
			if ed.HasPermission(structures.UserEditorPermissionModifyEmotes) {
				result = append(result, helpers.UserEditorStructureToModel(ed, r.Ctx.Config().CdnURL))
			}
		}
	}

	return result, err
}

func (r *Resolver) OwnedEmotes(ctx context.Context, obj *model.User) ([]*model.Emote, error) {
	emotes := []*structures.Emote{}
	errs := []error{}

	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmotes).Find(ctx, bson.M{
		"owner_id": obj.ID,
	})
	if err == nil {
		if err = cur.All(ctx, &emotes); err != nil {
			zap.S().Errorw("mongo, failed to retrieve user's owned emotes",
				"error", err,
			)

			errs = append(errs, errors.ErrUnknownEmote())
		}
	}

	result := make([]*model.Emote, len(emotes))

	for i, e := range emotes {
		if e == nil {
			continue
		}

		result[i] = helpers.EmoteStructureToModel(*e, r.Ctx.Config().CdnURL)
	}

	return result, multierror.Append(nil, errs...).ErrorOrNil()
}

func (r *Resolver) InboxUnreadCount(ctx context.Context, obj *model.User) (int, error) {
	// TODO
	return 0, nil
}

func (r *Resolver) Reports(ctx context.Context, obj *model.User) ([]*model.Report, error) {
	// TODO
	return nil, nil
}

func (r *Resolver) Activity(ctx context.Context, obj *model.User, limitArg *int) ([]*model.AuditLog, error) {
	result := []*model.AuditLog{}

	limit := 50
	if limitArg != nil {
		limit = *limitArg

		if limit > 300 {
			return result, errors.ErrInvalidRequest().SetDetail("limit must be less than 300")
		} else if limit < 1 {
			return result, errors.ErrInvalidRequest().SetDetail("limit must be greater than 0")
		}
	}

	// Fetch user's active emote sets
	sets := []primitive.ObjectID{}

	for _, con := range obj.Connections {
		if con.EmoteSetID.IsZero() {
			continue
		}

		sets = append(sets, *con.EmoteSetID)
	}

	logs := []structures.AuditLog{}
	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameAuditLogs).Find(ctx, bson.M{
		"$or": bson.A{
			bson.M{"target_id": obj.ID, "target_kind": structures.ObjectKindUser},
			bson.M{"target_id": bson.M{"$in": sets}, "target_kind": structures.ObjectKindEmoteSet},
		},
	}, options.Find().SetSort(bson.M{"_id": -1}).SetLimit(int64(limit)))

	if err != nil {
		zap.S().Errorw("mongo, failed to query user audit logs", "error", err)
		return result, errors.ErrInternalServerError()
	}

	if err := cur.All(ctx, &logs); err != nil {
		return result, errors.ErrInternalServerError()
	}

	actorMap := make(map[primitive.ObjectID]structures.User)

	for _, l := range logs {
		a := &model.AuditLog{
			ID:         l.ID,
			Kind:       int(l.Kind),
			ActorID:    l.ActorID,
			TargetID:   l.TargetID,
			TargetKind: int(l.TargetKind),
			CreatedAt:  l.ID.Timestamp(),
			Changes:    make([]*model.AuditLogChange, len(l.Changes)),
			Reason:     l.Reason,
		}

		actorMap[l.ActorID] = structures.DeletedUser

		// Append changes
		for i, c := range l.Changes {
			val := map[string]any{}
			aryval := model.AuditLogChangeArray{}

			switch c.Format {
			case structures.AuditLogChangeFormatSingleValue:
				_ = bson.Unmarshal(c.Value, &val)
			case structures.AuditLogChangeFormatArrayChange:
				_ = bson.Unmarshal(c.Value, &aryval)
			}

			a.Changes[i] = &model.AuditLogChange{
				Format:     int(c.Format),
				Key:        c.Key,
				Value:      val,
				ArrayValue: &aryval,
			}
		}

		result = append(result, a)
	}

	// Fetch and add actors to the result

	i := 0
	actorIDs := make([]primitive.ObjectID, len(actorMap))

	for oid := range actorMap {
		actorIDs[i] = oid
		i++
	}

	actors, errs := r.Ctx.Inst().Loaders.UserByID().LoadAll(actorIDs)
	if multierror.Append(nil, errs...).ErrorOrNil() != nil {
		return result, errors.ErrInternalServerError()
	}

	for _, u := range actors {
		actorMap[u.ID] = u
	}

	// Add actors to result
	for i, l := range result {
		result[i].Actor = helpers.UserStructureToPartialModel(helpers.UserStructureToModel(actorMap[l.ActorID], r.Ctx.Config().CdnURL))
	}

	return result, nil
}
