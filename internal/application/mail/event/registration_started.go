package event

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/mail"
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

	l := h.logger.With(
		slog.String("event", "RegistrationStarted"),
		slog.String("registration.id", e.RegistrationID.String()),
	)
	ctx, span := h.tracer.Start(
		ctx,
		"RegistrationStartedHandler.Handle",
		trace.WithNewRoot(),
		trace.WithLinks(trace.LinkFromContext(e.Extract())),
		trace.WithAttributes(attribute.String("event.registration.id", e.RegistrationID.String())),
	)
	defer span.End()

	if e.Email == "" {
		span.RecordError(errors.New("email is empty"))
		span.SetStatus(codes.Error, "email is empty")
		l.ErrorContext(ctx, "Email is empty")
		return errors.New("email is empty")
	}
	if e.VerificationCode == "" {
		span.RecordError(errors.New("verification code is empty"))
		span.SetStatus(codes.Error, "verification code is empty")
		l.ErrorContext(ctx, "Verification code is empty")
		return errors.New("verification code is empty")
	}

	payload := mail.Payload{
		To:      e.Email,
		Subject: "Email Verification Code",
		Body:    fmt.Sprintf("Your email verification code is: %s", e.VerificationCode),
	}
	if err := h.mailsender.SendMail(ctx, payload); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to send email verification code")
		l.ErrorContext(ctx, "Failed to send email verification code", slog.Any("error", err))
		return fmt.Errorf("failed to send email verification code: %w", err)
	}

	return nil
}
