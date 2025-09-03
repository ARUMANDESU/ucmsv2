package mailevent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/mail"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/logging"
	"github.com/ARUMANDESU/ucms/pkg/otelx"
)

const WelcomeSubject = "Welcome to UCMS"

func (h *MailEventHandler) HandleStudentRegistered(ctx context.Context, e *user.StudentRegistered) error {
	if e == nil {
		return nil
	}
	const op = "mailevent.MailEventHandler.HandleStudentRegistered"
	ctx, span := h.tracer.Start(ctx, "MailEventHandler.HandleStudentRegistered",
		trace.WithNewRoot(),
		trace.WithLinks(trace.LinkFromContext(e.Extract())),
		trace.WithAttributes(
			attribute.String("student.barcode", e.StudentBarcode.String()),
			attribute.String("student.email", logging.RedactEmail(e.Email)),
			attribute.String("student.group.id", e.GroupID.String())),
	)
	defer span.End()

	l := h.logger.With(
		slog.String("event", "StudentRegistered"),
		slog.String("student.barcode", e.StudentBarcode.String()),
		slog.String("student.email", logging.RedactEmail(e.Email)),
		slog.String("student.group.id", e.GroupID.String()))

	err := validation.ValidateStruct(e, validation.Field(&e.Email, validation.Required, is.EmailFormat))
	if err != nil {
		otelx.RecordSpanError(span, err, "invalid student registration data")
		l.ErrorContext(ctx, "invalid student registration data", "error", err.Error())
		return errorx.Wrap(err, op)
	}

	payload := mail.Payload{
		To:      e.Email,
		Subject: WelcomeSubject,
		Body: fmt.Sprintf(
			"Hello %s %s,\n\nWelcome to UCMS! Your registration is successful.\n\nBest regards,\nUCMS Team",
			e.FirstName,
			e.LastName,
		),
	}

	if err := h.mailsender.SendMail(ctx, payload); err != nil {
		otelx.RecordSpanError(span, err, "failed to send registration email")
		l.ErrorContext(ctx, "failed to send registration email", slog.Any("error", err))
		return errorx.Wrap(err, op)
	}

	return nil
}
