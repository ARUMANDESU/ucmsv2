package event

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/mail"
	"github.com/ARUMANDESU/ucms/pkg/logging"
	"github.com/ARUMANDESU/ucms/pkg/otelx"
)

var (
	tracer = otel.Tracer("ucms/application/mail/event")
	logger = otelslog.NewLogger("ucms/application/mail/event")
)

type MailSender interface {
	SendMail(ctx context.Context, payload mail.Payload) error
}

type RegistrationStartedHandler struct {
	tracer     trace.Tracer
	logger     *slog.Logger
	mailsender MailSender
}

type RegistrationStartedHandlerArgs struct {
	Tracer     trace.Tracer
	Logger     *slog.Logger
	Mailsender MailSender
}

func NewRegistrationStartedHandler(args RegistrationStartedHandlerArgs) *RegistrationStartedHandler {
	if args.Tracer == nil {
		args.Tracer = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}

	return &RegistrationStartedHandler{
		tracer:     args.Tracer,
		logger:     args.Logger,
		mailsender: args.Mailsender,
	}
}

func (h *RegistrationStartedHandler) Handle(ctx context.Context, e *registration.RegistrationStarted) error {
	if e == nil {
		return nil
	}

	l := h.logger.With(slog.String("event", "RegistrationStarted"), slog.String("registration.id", e.RegistrationID.String()))
	ctx, span := h.tracer.Start(
		ctx,
		"RegistrationStartedHandler.Handle",
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
		return err
	}

	payload := mail.Payload{
		To:      e.Email,
		Subject: "Email Verification Code",
		Body:    fmt.Sprintf("Your email verification code is: %s", e.VerificationCode),
	}
	if err := h.mailsender.SendMail(ctx, payload); err != nil {
		otelx.RecordSpanError(span, err, "failed to send email verification code")
		l.ErrorContext(ctx, "Failed to send email verification code", slog.Any("error", err))
		return fmt.Errorf("failed to send email verification code: %w", err)
	}

	return nil
}
