package event

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/mail"
)

type VerificationCodeResentHandler struct {
	tracer     trace.Tracer
	logger     *slog.Logger
	mailsender MailSender
}

type VerificationCodeResentHandlerArgs struct {
	Tracer     trace.Tracer
	Logger     *slog.Logger
	Mailsender MailSender
}

func NewVerificationCodeResentHandler(args VerificationCodeResentHandlerArgs) *VerificationCodeResentHandler {
	if args.Tracer == nil {
		args.Tracer = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}

	return &VerificationCodeResentHandler{
		tracer:     args.Tracer,
		logger:     args.Logger,
		mailsender: args.Mailsender,
	}
}

func (h *VerificationCodeResentHandler) Handle(ctx context.Context, e *registration.VerificationCodeResent) error {
	if e == nil {
		return nil
	}
	l := h.logger.With(
		slog.String("event", "VerificationCodeResent"),
		slog.String("registration.id", e.RegistrationID.String()),
	)
	ctx, span := h.tracer.Start(
		ctx,
		"VerificationCodeResentHandler.Handle",
		trace.WithNewRoot(),
		trace.WithLinks(trace.LinkFromContext(e.Extract())),
		trace.WithAttributes(attribute.String("event.registration.id", e.RegistrationID.String())),
	)
	defer span.End()

	l.DebugContext(ctx, "Handling VerificationCodeResent event by mail application",
		slog.String("email", e.Email),
		slog.String("verification_code", e.VerificationCode),
	)

	if err := h.mailsender.SendMail(ctx, mail.Payload{
		To:      e.Email,
		Subject: "Verification Code Resent",
		Body:    fmt.Sprintf("Your verification code has been resent: %s", e.VerificationCode),
	}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to send verification code resent email")
		h.logger.ErrorContext(ctx, "Failed to send verification code resent email", slog.Any("error", err))
		return err
	}

	return nil
}
