package event

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
)

var (
	tracer = otel.Tracer("ucms/internal/application/student/event")
	logger = otelslog.NewLogger("ucms/internal/application/student/event")
)

type Repo interface {
	SaveStudent(ctx context.Context, student *user.Student) error
}

type StudentRegistrationCompletedHandler struct {
	tracer      trace.Tracer
	logger      *slog.Logger
	studentrepo Repo
}

type StudentRegistrationCompletedHandlerArgs struct {
	Tracer      trace.Tracer
	Logger      *slog.Logger
	StudentRepo Repo
}

func NewStudentRegistrationCompletedHandler(args StudentRegistrationCompletedHandlerArgs) *StudentRegistrationCompletedHandler {
	if args.Tracer == nil {
		args.Tracer = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}

	return &StudentRegistrationCompletedHandler{
		tracer:      args.Tracer,
		logger:      args.Logger,
		studentrepo: args.StudentRepo,
	}
}

func (h *StudentRegistrationCompletedHandler) Handle(ctx context.Context, e *registration.StudentRegistrationCompleted) error {
	if e == nil {
		return nil
	}

	ctx, span := h.tracer.Start(ctx, "StudentRegistrationCompletedHandler.Handle",
		trace.WithNewRoot(),
		trace.WithLinks(trace.LinkFromContext(e.Extract())),
		trace.WithAttributes(attribute.String("event.registration_id", e.RegistrationID.String())),
	)
	defer span.End()

	h.logger.DebugContext(ctx, "handling student registration completed event", slog.Any("event", e))

	student, err := user.RegisterStudent(user.RegisterStudentArgs{
		ID:        user.ID(e.Barcode),
		FirstName: e.FirstName,
		LastName:  e.LastName,
		AvatarURL: "",
		Email:     e.Email,
		PassHash:  e.PassHash,
		GroupID:   e.GroupID,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to register student")
		h.logger.ErrorContext(ctx, "failed to register student")
		return err
	}

	if err := h.studentrepo.SaveStudent(ctx, student); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to save student")
		h.logger.ErrorContext(ctx, "failed to save student", slog.Any("student", student))
		return err
	}

	h.logger.DebugContext(ctx, "student registration completed")

	return nil
}
