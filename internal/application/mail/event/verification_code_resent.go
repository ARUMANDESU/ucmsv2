package mailevent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/mail"
	"github.com/ARUMANDESU/ucms/pkg/logging"
	"github.com/ARUMANDESU/ucms/pkg/otelx"
)

const VerificationCodeResentSubject = "Verification Code Resent"

func (h *MailEventHandler) HandleVerificationCodeResent(ctx context.Context, e *registration.VerificationCodeResent) error {
	if e == nil {
		return nil
	}
	l := h.logger.With(
		slog.String("event", "VerificationCodeResent"),
		slog.String("registration.id", e.RegistrationID.String()),
		slog.String("registration.email", logging.RedactEmail(e.Email)),
	)
	ctx, span := h.tracer.Start(
		ctx,
		"MailEventHandler.HandleVerificationCodeResent",
		trace.WithNewRoot(),
		trace.WithLinks(trace.LinkFromContext(e.Extract())),
		trace.WithAttributes(
			attribute.String("event.registration.id", e.RegistrationID.String()),
			attribute.String("event.registration.email", logging.RedactEmail(e.Email)),
		),
	)
	defer span.End()

	l.DebugContext(ctx, "Handling VerificationCodeResent event by mail application",
		slog.String("email", e.Email),
		slog.String("verification_code", e.VerificationCode),
	)

	err := validation.ValidateStruct(e,
		validation.Field(&e.Email, validation.Required, is.EmailFormat),
		validation.Field(&e.VerificationCode, validation.Required))
	if err != nil {
		otelx.RecordSpanError(span, err, "invalid verification code resent data")
		l.ErrorContext(ctx, "invalid verification code resent data", slog.Any("error", err))
		return err
	}

	if err := h.mailsender.SendMail(ctx, mail.Payload{
		To:      e.Email,
		Subject: VerificationCodeResentSubject,
		Body:    fmt.Sprintf("Your verification code has been resent: %s", e.VerificationCode),
	}); err != nil {
		otelx.RecordSpanError(span, err, "failed to send verification code resent email")
		h.logger.ErrorContext(ctx, "failed to send verification code resent email", slog.Any("error", err))
		return err
	}

	return nil
}
