package query

import (
	"context"

	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func (r *Resolver) Reports(ctx context.Context, statusArg *model.ReportStatus, limitArg *int, afterIDArg *primitive.ObjectID, beforeIDArg *primitive.ObjectID) ([]*model.Report, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	// Define limit
	limit := int64(12)
	if limitArg != nil {
		limit = int64(*limitArg)
	}

	if limit > 100 {
		limit = 100
	}

	// Paginate
	pagination := bson.M{}
	filter := bson.M{}

	if statusArg != nil {
		filter["status"] = *statusArg
	}

	if afterIDArg != nil {
		pagination["$gt"] = *afterIDArg
	}

	if beforeIDArg != nil {
		pagination["$lt"] = *beforeIDArg
	}

	if len(pagination) > 0 {
		filter["_id"] = pagination
	}

	opt := options.Find().SetLimit(limit).SetSort(bson.M{"created_at": 1})

	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameReports).Find(ctx, filter, opt)
	if err != nil {
		zap.S().Errorw("mongo, failed to create reports query", "error", err)

		return nil, errors.ErrInternalServerError()
	}

	reports := []structures.Report{}
	if err := cur.All(ctx, &reports); err != nil {
		zap.S().Errorw("mongo, failed to query reports")

		return nil, errors.ErrInternalServerError()
	}

	result := make([]*model.Report, len(reports))
	for i, report := range reports {
		result[i] = helpers.ReportStructureToModel(report)
	}

	// TODO
	return result, nil
}

func (r *Resolver) Report(ctx context.Context, id primitive.ObjectID) (*model.Report, error) {
	report := structures.Report{}
	if err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameReports).FindOne(ctx, bson.M{"_id": id}).Decode(&report); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrUnknownReport()
		}

		return nil, errors.ErrInternalServerError()
	}

	return helpers.ReportStructureToModel(report), nil
}
