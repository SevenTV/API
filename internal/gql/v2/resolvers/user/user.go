package user

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/internal/gql/v2/gen/generated"
	"github.com/seventv/api/internal/gql/v2/gen/model"
	"github.com/seventv/api/internal/gql/v2/helpers"
	"github.com/seventv/api/internal/gql/v2/types"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.UserResolver {
	return &Resolver{
		Resolver: r,
	}
}

func (r *Resolver) Role(ctx context.Context, obj *model.User) (*model.Role, error) {
	if obj.Role == nil {
		// Get default role
		roles, err := r.Ctx.Inst().Query.Roles(ctx, bson.M{"default": true})
		if err == nil && len(roles) > 0 {
			obj.Role = helpers.RoleStructureToModel(roles[0])
		} else {
			obj.Role = helpers.RoleStructureToModel(structures.NilRole)
		}
	}

	return obj.Role, nil
}

func (r *Resolver) Emotes(ctx context.Context, obj *model.User) ([]*model.Emote, error) {
	setID, err := primitive.ObjectIDFromHex(obj.EmoteSetID)
	if err != nil {
		return []*model.Emote{}, nil
	}

	set, err := r.Ctx.Inst().Loaders.EmoteSetByID().Load(setID)
	if err != nil {
		// send empty slice if no emote set
		return []*model.Emote{}, nil
	}

	emoteIDs := utils.Map(set.Emotes, func(a structures.ActiveEmote) primitive.ObjectID {
		return a.ID
	})

	emotes, _ := r.Ctx.Inst().Loaders.EmoteByID().LoadAll(emoteIDs)

	emoteMap := map[primitive.ObjectID]structures.Emote{}
	for _, emote := range emotes {
		emoteMap[emote.ID] = emote
	}

	arr := []*model.Emote{}

	for _, emote := range set.Emotes {
		if em, ok := emoteMap[emote.ID]; !ok {
			continue
		} else {
			emote.Emote = &em
		}

		ver, _ := emote.Emote.GetVersion(emote.ID)

		if ver.State.Lifecycle < structures.EmoteLifecycleLive {
			continue // skip if not live. non-live emotes can't be supported in v2
		}

		em := helpers.EmoteStructureToModel(*emote.Emote, r.Ctx.Config().CdnURL)

		// set "alias"
		if emote.Name != em.Name {
			em.OriginalName = &emote.Emote.Name
			em.Name = emote.Name
		}

		zw := emote.Emote.Flags.Value()&structures.EmoteFlagsZeroWidth != 0
		if zw && !utils.BitField.HasBits(int64(obj.Permissions), int64(structures.RolePermissionFeatureZeroWidthEmoteType)) {
			continue // omit zero-width if unprivileged
		}

		arr = append(arr, em)
	}

	return arr, nil
}

func (r *Resolver) EmoteIds(ctx context.Context, obj *model.User) ([]string, error) {
	setID, err := primitive.ObjectIDFromHex(obj.EmoteSetID)
	if err != nil {
		return []string{}, nil
	}

	result := []string{}

	set, err := r.Ctx.Inst().Loaders.EmoteSetByID().Load(setID)
	if err != nil {
		if errors.Compare(err, errors.ErrUnknownEmoteSet()) {
			return result, nil
		}

		return result, err
	}

	emoteIDs := utils.Map(set.Emotes, func(a structures.ActiveEmote) primitive.ObjectID {
		return a.ID
	})

	emotes, _ := r.Ctx.Inst().Loaders.EmoteByID().LoadAll(emoteIDs)

	emoteMap := map[primitive.ObjectID]structures.Emote{}
	for _, emote := range emotes {
		emoteMap[emote.ID] = emote
	}

	for _, e := range set.Emotes {
		if em, ok := emoteMap[e.ID]; !ok {
			continue
		} else {
			e.Emote = &em
		}

		ver, _ := e.Emote.GetVersion(e.ID)

		if ver.State.Lifecycle < structures.EmoteLifecycleLive {
			continue // skip if not live. non-live emotes can't be supported in v2
		}

		if e.Emote != nil {
			zw := e.Emote.Flags.Value()&structures.EmoteFlagsZeroWidth != 0
			if zw && !utils.BitField.HasBits(int64(obj.Permissions), int64(structures.RolePermissionFeatureZeroWidthEmoteType)) {
				continue // omit zero-width if unprivileged
			}
		}

		result = append(result, e.ID.Hex())
	}

	return result, nil
}

func (r *Resolver) EmoteAliases(ctx context.Context, obj *model.User) ([][]string, error) {
	setID, err := primitive.ObjectIDFromHex(obj.EmoteSetID)
	if err != nil {
		return [][]string{}, nil
	}

	result := [][]string{}

	set, err := r.Ctx.Inst().Loaders.EmoteSetByID().Load(setID)
	if err != nil {
		// send empty slice if no emote set
		return [][]string{}, nil
	}

	emoteIDs := utils.Map(set.Emotes, func(a structures.ActiveEmote) primitive.ObjectID {
		return a.ID
	})

	emotes, _ := r.Ctx.Inst().Loaders.EmoteByID().LoadAll(emoteIDs)

	emoteMap := map[primitive.ObjectID]structures.Emote{}
	for _, emote := range emotes {
		emoteMap[emote.ID] = emote
	}

	for _, e := range set.Emotes {
		if em, ok := emoteMap[e.ID]; !ok {
			continue
		} else {
			e.Emote = &em
		}

		if e.Name == e.Emote.Name {
			continue // no original name property means no alias set
		}

		result = append(result, []string{e.ID.Hex(), e.Name})
	}

	return result, nil
}

func (r *Resolver) Editors(ctx context.Context, obj *model.User) ([]*model.UserPartial, error) {
	var err error

	result := []*model.UserPartial{}
	editorIDs := make([]primitive.ObjectID, len(obj.EditorIds))

	for i, v := range obj.EditorIds {
		editorIDs[i], err = primitive.ObjectIDFromHex(v)
		if err != nil {
			return nil, errors.ErrBadObjectID()
		}
	}

	editors, errs := r.Ctx.Inst().Loaders.UserByID().LoadAll(editorIDs)
	if err := multierror.Append(nil, errs...).ErrorOrNil(); err != nil {
		return result, err
	}

	for _, ed := range editors {
		result = append(result, helpers.UserStructureToPartialModel(helpers.UserStructureToModel(ed, r.Ctx.Config().CdnURL)))
	}

	return result, nil
}

func (r *Resolver) EditorIn(ctx context.Context, obj *model.User) ([]*model.UserPartial, error) {
	result := []*model.UserPartial{}

	userID, err := primitive.ObjectIDFromHex(obj.ID)
	if err != nil {
		return result, err
	}

	editors, err := r.Ctx.Inst().Query.UserEditorOf(ctx, userID)
	if err != nil {
		return result, err
	}

	// Get a list of user IDs from the v3 editor list
	ids := make([]primitive.ObjectID, len(editors))
	for i, ed := range editors {
		ids[i] = ed.ID
	}

	users, errs := r.Ctx.Inst().Loaders.UserByID().LoadAll(ids)
	if err = multierror.Append(nil, errs...).ErrorOrNil(); err != nil {
		return result, err
	}

	for _, u := range users {
		result = append(result, helpers.UserStructureToPartialModel(helpers.UserStructureToModel(u, r.Ctx.Config().CdnURL)))
	}

	return result, nil
}

func (r *Resolver) Notifications(ctx context.Context, obj *model.User) ([]*model.Notification, error) {
	return []*model.Notification{{
		ID:           primitive.NewObjectID().Hex(),
		Announcement: true,
		Title:        "Notifications have evolved",
		Timestamp:    time.Now().Format(time.RFC3339),
		MessageParts: []*model.NotificationMessagePart{{
			Type: 1,
			Data: fmt.Sprintf("The new Inbox system replaces notifications! To see your messages, go to %s", r.Ctx.Config().WebsiteURL),
		}},
		Read:   false,
		ReadAt: new(string),
	}}, nil
}

func (r *Resolver) Cosmetics(ctx context.Context, obj *model.User) ([]*model.UserCosmetic, error) {
	cosmetics := []structures.Cosmetic[bson.Raw]{}

	oid, err := primitive.ObjectIDFromHex(obj.ID)
	if err != nil {
		return nil, errors.ErrBadObjectID()
	}

	pipeline := mongo.Pipeline{
		{{
			Key: "$match",
			Value: bson.M{
				"user_id": oid,
			},
		}},
		{{
			Key: "$lookup",
			Value: bson.M{
				"from":         "cosmetics",
				"localField":   "data.ref",
				"foreignField": "_id",
				"as":           "cosmetic",
			},
		}},
		{{Key: "$set", Value: bson.M{"cosmetic": bson.M{"$first": "$cosmetic"}}}},
		{{
			Key: "$project",
			Value: bson.M{
				"_id":      "$cosmetic._id",
				"kind":     "$cosmetic.kind",
				"name":     "$cosmetic.name",
				"data":     "$cosmetic.data",
				"selected": "$data.selected",
			},
		}},
	}

	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEntitlements).Aggregate(ctx, pipeline)
	if err != nil {
		return []*model.UserCosmetic{}, errors.ErrInternalServerError()
	}

	if err := cur.All(ctx, &cosmetics); err != nil {
		return []*model.UserCosmetic{}, errors.ErrInternalServerError()
	}

	result := make([]*model.UserCosmetic, len(cosmetics))
	for i, cos := range cosmetics {
		result[i] = helpers.CosmeticStructureToModel(cos)
	}

	return result, nil
}

// OwnedEmotes implements generated.UserResolver
func (r *Resolver) OwnedEmotes(ctx context.Context, obj *model.User) ([]*model.Emote, error) {
	oid, err := primitive.ObjectIDFromHex(obj.ID)
	if err != nil {
		return []*model.Emote{}, errors.ErrBadObjectID()
	}

	emotes, err := r.Ctx.Inst().Loaders.EmoteByOwnerID().Load(oid)
	if err != nil {
		if errors.Compare(err, errors.ErrNoItems()) {
			return []*model.Emote{}, nil
		}

		return []*model.Emote{}, err
	}

	result := make([]*model.Emote, len(emotes))
	for i, e := range emotes {
		result[i] = helpers.EmoteStructureToModel(e, r.Ctx.Config().CdnURL)
	}

	return result, nil
}
