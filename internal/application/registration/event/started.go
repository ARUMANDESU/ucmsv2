package event

import (
	"context"
	"errors"
	"fmt"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
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
	mailsender MailSender
}

type RegistrationStartedHandlerArgs struct {
	Mailsender MailSender
}

func NewRegistrationStartedHandler(args RegistrationStartedHandlerArgs) *RegistrationStartedHandler {
	return &RegistrationStartedHandler{
		mailsender: args.Mailsender,
	}
}

func (h *RegistrationStartedHandler) Handle(ctx context.Context, e *registration.RegistrationStarted) error {
	if e == nil {
		return errors.New("event is nil")
	}
	if e.Email == "" {
		return errors.New("email is empty")
	}
	if e.VerificationCode == "" {
		return errors.New("verification code is empty")
	}

	payload := MailSenderPayload{
		ToEmail: e.Email,
		Subject: "Email Verification Code",
		Message: fmt.Sprintf("Your email verification code is: %s", e.VerificationCode),
	}
	if err := h.mailsender.MailSend(ctx, payload); err != nil {
		return fmt.Errorf("failed to send email verification code: %w", err)
	}

	return nil
}
