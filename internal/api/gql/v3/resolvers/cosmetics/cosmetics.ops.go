package cosmetics

import (
	"context"

	"github.com/seventv/api/internal/api/gql/v3/gen/generated"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/api/internal/api/gql/v3/types"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type ResolverOps struct {
	types.Resolver
}

func NewOps(r types.Resolver) generated.CosmeticOpsResolver {
	return &ResolverOps{r}
}

// Paint implements generated.CosmeticOpsResolver
func (r *ResolverOps) UpdatePaint(ctx context.Context, obj *model.CosmeticOps, def model.CosmeticPaintInput) (*model.CosmeticPaint, error) {
	mainColor := 0
	if def.Color != nil {
		mainColor = *def.Color
	}

	angle := 90
	if def.Angle != nil {
		angle = *def.Angle
	}

	shape := ""
	if def.Shape != nil {
		shape = *def.Shape
	}

	imgURL := ""
	if def.ImageURL != nil {
		imgURL = *def.ImageURL
	}

	stops := make([]structures.CosmeticPaintGradientStop, len(def.Stops))
	for i, st := range def.Stops {
		stops[i] = structures.CosmeticPaintGradientStop{
			At:    st.At,
			Color: utils.Color(st.Color),
		}
	}

	shadows := make([]structures.CosmeticPaintDropShadow, len(def.Shadows))
	for i, sh := range def.Shadows {
		shadows[i] = structures.CosmeticPaintDropShadow{
			OffsetX: sh.XOffset,
			OffsetY: sh.YOffset,
			Radius:  sh.Radius,
			Color:   utils.Color(sh.Color),
		}
	}

	cos := structures.Cosmetic[structures.CosmeticDataPaint]{
		ID:       obj.ID,
		Kind:     structures.CosmeticKindNametagPaint,
		Priority: 0,
		Name:     def.Name,
		Data: structures.CosmeticDataPaint{
			Function:    structures.CosmeticPaintFunction(def.Function),
			Color:       utils.PointerOf(utils.Color(mainColor)),
			Stops:       stops,
			Repeat:      def.Repeat,
			Angle:       int32(angle),
			Shape:       shape,
			ImageURL:    imgURL,
			DropShadows: shadows,
		},
	}

	// Update the cosmetic in DB
	result := structures.Cosmetic[structures.CosmeticDataPaint]{}
	if err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameCosmetics).FindOneAndUpdate(ctx, bson.M{
		"_id": obj.ID,
	}, bson.M{
		"$set": cos,
	}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&result); err != nil {
		zap.S().Errorw("failed to update cosmetic", "cosmetic", cos.ID.Hex(), "error", err)
		return nil, err
	}

	return r.Ctx.Inst().Modelizer.Paint(result).GQL(), nil
}
