package event

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/mail"
	"github.com/ARUMANDESU/ucms/pkg/logging"
)

type StudentRegisteredHandler struct {
	tracer     trace.Tracer
	logger     *slog.Logger
	mailsender MailSender
}

type StudentRegisteredHandlerArgs struct {
	Tracer     trace.Tracer
	Logger     *slog.Logger
	Mailsender MailSender
}

func NewStudentRegisteredHandler(args StudentRegisteredHandlerArgs) *StudentRegisteredHandler {
	if args.Tracer == nil {
		args.Tracer = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}

	return &StudentRegisteredHandler{
		tracer:     args.Tracer,
		logger:     args.Logger,
		mailsender: args.Mailsender,
	}
}

func (h *StudentRegisteredHandler) Handle(ctx context.Context, e *user.StudentRegistered) error {
	if e == nil {
		return nil
	}

	l := h.logger.With(
		slog.String("event", "StudentRegistered"),
		slog.String("student.barcode", e.StudentBarcode.String()),
		slog.String("student.email", logging.RedactEmail(e.Email)),
		slog.String("student.group.id", e.GroupID.String()))

	ctx, span := h.tracer.Start(
		ctx,
		"StudentRegisteredHandler.Handle",
		trace.WithNewRoot(),
		trace.WithLinks(trace.LinkFromContext(e.Extract())),
		trace.WithAttributes(
			attribute.String("student.barcode", e.StudentBarcode.String()),
			attribute.String("student.email", logging.RedactEmail(e.Email)),
			attribute.String("student.group.id", e.GroupID.String())),
	)
	defer span.End()

	err := validation.ValidateStruct(e, validation.Field(&e.Email, validation.Required, is.EmailFormat))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid student registration data")
		l.ErrorContext(ctx, "invalid student registration data", "error", err.Error())
		return err
	}

	payload := mail.Payload{
		To:      e.Email,
		Subject: "Welcome to UCMS",
		Body: fmt.Sprintf(
			"Hello %s %s,\n\nWelcome to UCMS! Your registration is successful.\n\nBest regards,\nUCMS Team",
			e.FirstName,
			e.LastName,
		),
	}

	if err := h.mailsender.SendMail(ctx, payload); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to send registration email")
		l.ErrorContext(ctx, "failed to send registration email", slog.Any("error", err))
		return err
	}

	return nil
}
