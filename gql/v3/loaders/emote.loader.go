package loaders

import (
	"context"
	"time"

	"github.com/SevenTV/Common/dataloader"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/seventv/api/global"
	"github.com/seventv/api/gql/v3/gen/model"
	"github.com/seventv/api/gql/v3/helpers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func emoteByID(gCtx global.Context) *EmoteLoader {
	return dataloader.New(dataloader.Config[primitive.ObjectID, *model.Emote]{
		Fetch: func(keys []primitive.ObjectID) ([]*model.Emote, []error) {
			ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
			defer cancel()

			// Fetch emote data from the database
			models := make([]*model.Emote, len(keys))
			errs := make([]error, len(keys))

			// Initially fill the response with unknown emotes in case some cannot be found
			unknownModel := helpers.EmoteStructureToModel(gCtx, structures.DeletedEmote)
			for i := 0; i < len(models); i++ {
				models[i] = unknownModel
			}

			// Get roles (to assign to emote owners)
			roles, _ := gCtx.Inst().Query.Roles(ctx, bson.M{})
			roleMap := make(map[primitive.ObjectID]structures.Role)
			for _, role := range roles {
				roleMap[role.ID] = role
			}

			// Iterate over cursor
			// Transform emote structures into models
			emotes, err := gCtx.Inst().Query.Emotes(ctx, bson.M{
				"versions.id": bson.M{"$in": keys},
			}).Items()

			if err == nil {
				m := make(map[primitive.ObjectID]structures.Emote)
				for _, e := range emotes {
					for _, ver := range e.Versions {
						m[ver.ID] = e
					}
				}

				for i, v := range keys {
					if x, ok := m[v]; ok {
						ver, _ := x.GetVersion(v)
						if ver.IsUnavailable() {
							continue
						}
						x.ID = v
						models[i] = helpers.EmoteStructureToModel(gCtx, x)
					}
				}
			}

			return models, errs
		},
		Wait: time.Millisecond * 5,
	})
}
