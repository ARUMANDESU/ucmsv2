package mailevent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/registration"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/valueobject/mails"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/logging"
	"gitlab.com/ucmsv2/ucms-backend/pkg/otelx"
)

const RegistrationStartedSubject = "Email Verification Code"

func (h *MailEventHandler) HandleRegistrationStarted(ctx context.Context, e *registration.RegistrationStarted) error {
	if e == nil {
		return nil
	}
	const op = "mailevent.MailEventHandler.HandleRegistrationStarted"

	l := h.logger.With(slog.String("event", "RegistrationStarted"), slog.String("registration.id", e.RegistrationID.String()))
	ctx, span := h.tracer.Start(
		ctx,
		"MailEventHandler.HandleRegistrationStarted",
		trace.WithNewRoot(),
		trace.WithLinks(trace.LinkFromContext(e.Extract())),
		trace.WithAttributes(
			attribute.String("event.registration.id", e.RegistrationID.String()),
			attribute.String("event.registration.email", logging.RedactEmail(e.Email)),
		),
	)
	defer span.End()

	err := validation.ValidateStruct(e,
		validation.Field(&e.Email, validation.Required, is.EmailFormat),
		validation.Field(&e.VerificationCode, validation.Required),
	)
	if err != nil {
		otelx.RecordSpanError(span, err, "validation failed")
		l.ErrorContext(ctx, "validation failed", slog.Any("error", err))
		return errorx.Wrap(err, op)
	}

	payload := mails.Payload{
		To:      e.Email,
		Subject: RegistrationStartedSubject,
		Body:    fmt.Sprintf("Your email verification code is: %s", e.VerificationCode),
	}
	if err := h.mailsender.SendMail(ctx, payload); err != nil {
		otelx.RecordSpanError(span, err, "failed to send email verification code")
		l.ErrorContext(ctx, "Failed to send email verification code", slog.Any("error", err))
		return errorx.Wrap(err, op)
	}

	return nil
}
