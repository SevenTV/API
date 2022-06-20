package mutation

import (
	"context"
	"time"

	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// CreateCosmeticPaint implements generated.MutationResolver
func (r *Resolver) CreateCosmeticPaint(ctx context.Context, def model.CosmeticPaintInput) (primitive.ObjectID, error) {
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
			Color: int32(st.Color),
		}
	}

	shadows := make([]structures.CosmeticPaintDropShadow, len(def.Shadows))
	for i, sh := range def.Shadows {
		shadows[i] = structures.CosmeticPaintDropShadow{
			OffsetX: sh.XOffset,
			OffsetY: sh.YOffset,
			Radius:  sh.Radius,
			Color:   int32(sh.Color),
		}
	}

	cos := structures.Cosmetic[structures.CosmeticDataPaint]{
		ID:       primitive.NewObjectIDFromTimestamp(time.Now()),
		Kind:     structures.CosmeticKindNametagPaint,
		Priority: 0,
		Name:     def.Name,
		Data: structures.CosmeticDataPaint{
			Function:    structures.CosmeticPaintFunction(def.Function),
			Color:       utils.PointerOf(int32(mainColor)),
			Stops:       stops,
			Repeat:      def.Repeat,
			Angle:       int32(angle),
			Shape:       shape,
			ImageURL:    imgURL,
			DropShadows: shadows,
		},
	}

	result, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameCosmetics).InsertOne(ctx, cos)
	if err != nil {
		zap.S().Errorw("failed to create new paint cosmetic",
			"error", err,
		)
		return primitive.NilObjectID, errors.ErrInternalServerError().SetDetail(err.Error())
	}

	return result.InsertedID.(primitive.ObjectID), nil
}
