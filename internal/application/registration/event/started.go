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
)

var (
	tracer = otel.Tracer("ucms/application/registration/event")
	logger = otelslog.NewLogger("ucms/application/registration/event")
)

type MailSender interface {
	MailSend(ctx context.Context, payload MailSenderPayload) error
}

type MailSenderPayload struct {
	ToEmail string
	Subject string
	Message string
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
		return errors.New("event is nil")
	}

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
		return errors.New("email is empty")
	}
	if e.VerificationCode == "" {
		span.RecordError(errors.New("verification code is empty"))
		span.SetStatus(codes.Error, "verification code is empty")
		return errors.New("verification code is empty")
	}

	payload := MailSenderPayload{
		ToEmail: e.Email,
		Subject: "Email Verification Code",
		Message: fmt.Sprintf("Your email verification code is: %s", e.VerificationCode),
	}
	if err := h.mailsender.MailSend(ctx, payload); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to send email verification code")
		return fmt.Errorf("failed to send email verification code: %w", err)
	}

	return nil
}
