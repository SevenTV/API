package mutation

import (
	"context"
	"fmt"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

const (
	REPORT_SUBJECT_MIN_LENGTH      = 4
	REPORT_SUBJECT_MAX_LENGTH      = 72
	REPORT_BODY_MIN_LENGTH         = 0
	REPORT_BODY_MAX_LENGTH         = 2000
	REPORT_ALLOWED_ACTIVE_PER_USER = 3
)

func (r *Resolver) CreateReport(ctx context.Context, data model.CreateReportInput) (*model.Report, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	// Get and verify the target
	var errType error

	kind := structures.ObjectKind(data.TargetKind)

	switch structures.ObjectKind(data.TargetKind) {
	case structures.ObjectKindUser:
		errType = errors.ErrUnknownUser()
	case structures.ObjectKindEmote:
		errType = errors.ErrUnknownEmote()
	default:
		return nil, errors.ErrEmoteNameInvalid().SetDetail("You cannot report type %s", kind.String())
	}

	if c, _ := r.Ctx.Inst().Mongo.Collection(mongo.CollectionName(kind.CollectionName())).CountDocuments(ctx, bson.M{
		"_id": data.TargetID,
	}); c == 0 {
		return nil, errType
	}

	// Validate the input
	if len(data.Subject) < REPORT_SUBJECT_MIN_LENGTH {
		graphql.AddError(ctx, errors.ErrInvalidRequest().SetDetail(fmt.Sprintf("subject must be at least %d characters long", REPORT_SUBJECT_MIN_LENGTH)))
	}

	if len(data.Subject) > REPORT_SUBJECT_MAX_LENGTH {
		graphql.AddError(ctx, errors.ErrInvalidRequest().SetDetail(fmt.Sprintf("subject must be at most %d characters long", REPORT_SUBJECT_MAX_LENGTH)))
	}

	if len(data.Body) < REPORT_BODY_MIN_LENGTH {
		graphql.AddError(ctx, errors.ErrInvalidRequest().SetDetail(fmt.Sprintf("body must be at least %d characters long", REPORT_BODY_MIN_LENGTH)))
	}

	if len(data.Body) > REPORT_BODY_MAX_LENGTH {
		graphql.AddError(ctx, errors.ErrInvalidRequest().SetDetail(fmt.Sprintf("body must be at most %d characters long", REPORT_BODY_MAX_LENGTH)))
	}

	if len(graphql.GetErrors(ctx)) > 0 {
		return nil, errors.ErrValidationRejected().SetDetail("Some fields have been filled incorrectly")
	}

	// Create the report
	rb := structures.NewReportBuilder(structures.Report{
		AssigneeIDs: []primitive.ObjectID{},
	})
	rb.Report.ID = primitive.NewObjectIDFromTimestamp(time.Now())
	rb.SetTargetKind(kind).
		SetTargetID(data.TargetID).
		SetReporterID(actor.ID).
		SetStatus(structures.ReportStatusOpen).
		SetSubject(data.Subject).
		SetBody(data.Body).
		SetCreatedAt(time.Now())

	_, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameReports).InsertOne(ctx, rb.Report)
	if err != nil {
		zap.S().Errorw("mongo", "error", err)

		return nil, errors.ErrInternalServerError().SetDetail("Report creation could not be completed")
	}

	return &model.Report{}, nil
}

func (r *Resolver) EditReport(ctx context.Context, reportID primitive.ObjectID, data model.EditReportInput) (*model.Report, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	// Get the report
	report := structures.Report{}
	if err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameReports).FindOne(ctx, bson.M{"_id": reportID}).Decode(&report); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrUnknownReport()
		}

		return nil, errors.ErrInternalServerError()
	}

	// Apply mutations
	rb := structures.NewReportBuilder(report)

	if data.Priority != nil {
		rb.SetPriority(int32(*data.Priority))
	}

	if data.Status != nil {
		rb.SetStatus(structures.ReportStatus(*data.Status))
	}

	if data.Assignee != nil {
		a := *data.Assignee
		assigneeID, err := primitive.ObjectIDFromHex(a[1:])
		if err != nil {
			return nil, errors.ErrBadObjectID()
		}

		state := a[0]
		switch state {
		case '+':
			rb.AddAssignee(assigneeID)
		case '-':
			rb.RemoveAssignee(assigneeID)
		default:
			return nil, errors.ErrInvalidRequest().SetDetail("assignee must be prefixed with '+' or '-'")
		}
	}

	// Write
	if _, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameReports).UpdateOne(ctx, bson.M{
		"_id": reportID,
	}, rb.Update); err != nil {
		return nil, errors.ErrInternalServerError()
	}

	return helpers.ReportStructureToModel(rb.Report), nil
}
