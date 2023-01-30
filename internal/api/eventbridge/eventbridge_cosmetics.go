package eventbridge

import (
	"context"
	"strings"

	"github.com/seventv/api/data/events"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func handleCosmetics(gctx global.Context, ctx context.Context, body events.CosmeticsCommandBody) error {
	var sid string
	switch v := ctx.Value(SESSION_ID_KEY).(type) {
	case string:
		sid = v
	}

	// Identify the target
	idsp := strings.SplitN(body.Identifier, ":", 2)
	if len(idsp) != 2 {
		return errors.ErrInvalidRequest().SetDetail("Invalid Identifier Format")
	}

	idType := idsp[0]
	identifier := idsp[1]

	var (
		user structures.User
		err  error
	)

	// Platform specified: find by connection
	if body.Platform != "" {
		switch idType {
		case "id":
			user, err = gctx.Inst().Loaders.UserByConnectionID(body.Platform).Load(identifier)
		case "username":
			user, err = gctx.Inst().Loaders.UserByConnectionUsername(body.Platform).Load(identifier)
		}
	} else { // no platform means app user
		switch idType {
		case "id":
			oid, er := primitive.ObjectIDFromHex(identifier)
			if er != nil {
				err = er
				break
			}

			user, err = gctx.Inst().Loaders.UserByID().Load(oid)
		case "username":
			user, err = gctx.Inst().Loaders.UserByUsername().Load(identifier)
		}
	}

	if err != nil {
		return err
	}

	kinds := utils.Set[structures.CosmeticKind]{}
	kinds.Fill(body.Kinds...)

	// Dispatch user avatar
	if kinds.Has(structures.CosmeticKindAvatar) {
		av := gctx.Inst().Modelizer.Avatar(user)

		if _, err := gctx.Inst().Events.DispatchWithEffect(ctx, events.EventTypeCreateCosmetic, events.ChangeMap{
			ID:         user.ID,
			Kind:       structures.ObjectKindCosmetic,
			Contextual: true,
			Object:     utils.ToJSON(av),
		}, events.DispatchOptions{
			Whisper: sid,
		}); err != nil {
			return err
		}
	}

	return nil
}
