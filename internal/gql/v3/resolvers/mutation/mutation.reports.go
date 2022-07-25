package mutation

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/mutations"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

const (
	REPORT_SUBJECT_MIN_LENGTH      = 4
	REPORT_SUBJECT_MAX_LENGTH      = 72
	REPORT_BODY_MIN_LENGTH         = 4
	REPORT_BODY_MAX_LENGTH         = 2000
	REPORT_ALLOWED_ACTIVE_PER_USER = 3
)

func (r *Resolver) CreateReport(ctx context.Context, data model.CreateReportInput) (*model.Report, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	// Get and verify the target
	var (
		errType error
		kind    = structures.ObjectKind(data.TargetKind)
	)

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
	t := time.Now()

	l := kind.String()[:1]
	yr := strconv.Itoa(t.Year())
	mo := strconv.Itoa(int(t.Month()))
	dy := strconv.Itoa(t.Day())
	sc := int(time.Since(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)).Seconds())

	caseID := fmt.Sprintf("%s-%s%s%s%d", l, yr[len(yr)-2:], mo, dy, sc)

	rb := structures.NewReportBuilder(structures.Report{
		CaseID:      caseID,
		AssigneeIDs: []primitive.ObjectID{},
	})
	rb.Report.ID = primitive.NewObjectIDFromTimestamp(time.Now())
	rb.SetTargetKind(kind).
		SetTargetID(data.TargetID).
		SetReporterID(actor.ID).
		SetStatus(structures.ReportStatusOpen).
		SetSubject(data.Subject).
		SetBody(data.Body).
		SetCreatedAt(t)

	_, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameReports).InsertOne(ctx, rb.Report)
	if err != nil {
		zap.S().Errorw("mongo", "error", err)

		return nil, errors.ErrInternalServerError().SetDetail("Report creation could not be completed")
	}

	// Create AuditLog
	truncBody := data.Subject
	if len(truncBody) > 128 {
		truncBody = truncBody[:128] + "..."
	}

	alb := structures.NewAuditLogBuilder(structures.AuditLog{
		Reason: data.Subject + ": " + truncBody,
	}).
		SetActor(actor.ID).
		SetKind(structures.AuditLogKindCreateReport).
		SetTargetKind(structures.ObjectKindReport).
		SetTargetID(rb.Report.ID)

	if _, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameAuditLogs).InsertOne(ctx, alb.AuditLog); err != nil {
		zap.S().Errorw("mongo, failed to write audit log", "error", err)
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

	alb := structures.NewAuditLogBuilder(structures.AuditLog{}).
		SetKind(structures.AuditLogKindCreateReport).
		SetActor(actor.ID).
		SetTargetKind(structures.ObjectKindReport).
		SetTargetID(report.ID)

	if data.Priority != nil {
		p := *data.Priority

		alb = alb.AddChanges((&structures.AuditLogChange{
			Format: structures.AuditLogChangeFormatSingleValue,
			Key:    "priority",
		}).WriteSingleValues(report.Priority, p))

		rb.SetPriority(int32(p))
	}

	if data.Status != nil {
		st := *data.Status

		alb = alb.AddChanges((&structures.AuditLogChange{
			Format: structures.AuditLogChangeFormatSingleValue,
			Key:    "status",
		}).WriteSingleValues(report.Status, structures.ReportStatus(st)))

		rb.SetStatus(structures.ReportStatus(st))

		if st == model.ReportStatusClosed {
			rb.SetClosedAt(time.Now())

			// Send notification to the user that their report has been handled
			mb := structures.NewMessageBuilder(structures.Message[structures.MessageDataInbox]{}).
				SetKind(structures.MessageKindInbox).
				SetAuthorID(actor.ID).
				SetTimestamp(time.Now()).
				SetAnonymous(false).
				SetData(structures.MessageDataInbox{
					Subject: "inbox.generic.report_closed.subject",
					Content: "inbox.generic.report_closed.content",
					Locale:  true,
					System:  true,
					Placeholders: map[string]string{
						"CASE_ID": report.CaseID,
					},
				})

			_ = r.Ctx.Inst().Mutate.SendInboxMessage(ctx, mb, mutations.SendInboxMessageOptions{
				Actor:                &actor,
				Recipients:           []primitive.ObjectID{actor.ID},
				ConsiderBlockedUsers: false,
			})
		} else {
			rb.SetClosedAt(time.Time{})
		}
	}

	if data.Assignee != nil {
		a := *data.Assignee

		c := &structures.AuditLogChange{
			Format: structures.AuditLogChangeFormatArrayChange,
			Key:    "assignee_ids",
		}

		assigneeID, err := primitive.ObjectIDFromHex(a[1:])

		if err != nil {
			return nil, errors.ErrBadObjectID()
		}

		state := a[0]
		switch state {
		case '+':
			rb.AddAssignee(assigneeID)
			c.WriteArrayAdded(assigneeID)
		case '-':
			rb.RemoveAssignee(assigneeID)
			c.WriteArrayRemoved(assigneeID)
		default:
			return nil, errors.ErrInvalidRequest().SetDetail("assignee must be prefixed with '+' or '-'")
		}

		alb = alb.AddChanges(c)
	}

	rb.SetLastUpdatedAt(time.Now())

	// Write update
	if _, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameReports).UpdateOne(ctx, bson.M{
		"_id": reportID,
	}, rb.Update); err != nil {
		return nil, errors.ErrInternalServerError()
	}

	// Write audit log
	if _, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameAuditLogs).InsertOne(ctx, alb.AuditLog); err != nil {
		zap.S().Errorw("mongo, failed to write audit log", "error", err)
	}

	return helpers.ReportStructureToModel(rb.Report), nil
}
