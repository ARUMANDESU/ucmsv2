package event

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/staffinvitation"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/mail"
	"github.com/ARUMANDESU/ucms/pkg/logging"
	"github.com/ARUMANDESU/ucms/pkg/otelx"
)

func (h *MailEventHandler) HandleStaffInvitationCreated(ctx context.Context, e *staffinvitation.Created) error {
	if e == nil {
		return nil
	}
	ctx, span := h.tracer.Start(ctx, "MailEventHandler.HandleStaffInvitationCreated",
		trace.WithNewRoot(),
		trace.WithLinks(trace.LinkFromContext(e.Extract())),
		trace.WithAttributes(
			attribute.String("invitation.id", e.StaffInvitationID.String()),
			attribute.Int("invitation.recipients_email_count", len(e.RecipientsEmail)),
		),
	)
	defer span.End()

	l := h.logger.With(
		slog.String("event", "StaffInvitationCreated"),
		slog.String("invitation.id", e.StaffInvitationID.String()),
		slog.Int("invitation.recipients_email_count", len(e.RecipientsEmail)),
	)

	if len(e.RecipientsEmail) == 0 {
		l.DebugContext(ctx, "No recipient emails provided for staff invitation")
		return nil
	}

	for _, email := range e.RecipientsEmail {
		if err := h.sendStaffInvitationEmail(ctx, email, e.Code); err != nil {
			otelx.RecordSpanError(span, err, "failed to send staff invitation email")
			l.ErrorContext(ctx, "failed to send staff invitation email",
				slog.String("email", logging.RedactEmail(email)),
				slog.String("error", err.Error()),
			)
			// Continue sending emails to other recipients even if one fails
		}
	}

	return nil
}

func (h *MailEventHandler) HandleStaffInvitationRecipientsUpdated(ctx context.Context, e *staffinvitation.RecipientsUpdated) error {
	if e == nil {
		return nil
	}
	ctx, span := h.tracer.Start(ctx, "MailEventHandler.HandleStaffInvitationRecipientsUpdated",
		trace.WithNewRoot(),
		trace.WithLinks(trace.LinkFromContext(e.Extract())),
		trace.WithAttributes(
			attribute.String("invitation.id", e.StaffInvitationID.String()),
			attribute.Int("invitation.new_recipients_email_count", len(e.NewRecipientsEmail)),
		),
	)
	defer span.End()

	l := h.logger.With(
		slog.String("event", "StaffInvitationRecipientsUpdated"),
		slog.String("invitation.id", e.StaffInvitationID.String()),
		slog.Int("invitation.new_recipients_email_count", len(e.NewRecipientsEmail)),
	)

	if len(e.NewRecipientsEmail) == 0 {
		l.DebugContext(ctx, "No new recipient emails provided for staff invitation update")
		return nil
	}

	for _, email := range e.NewRecipientsEmail {
		if err := h.sendStaffInvitationEmail(ctx, email, e.Code); err != nil {
			otelx.RecordSpanError(span, err, "failed to send updated staff invitation email")
			l.ErrorContext(ctx, "failed to send updated staff invitation email",
				slog.String("email", logging.RedactEmail(email)),
				slog.String("error", err.Error()),
			)
			// Continue sending emails to other recipients even if one fails
		}
	}

	return nil
}

func (h *MailEventHandler) sendStaffInvitationEmail(ctx context.Context, email, code string) error {
	payload := mail.Payload{
		To:      email,
		Subject: "Staff Invitation",
		Body: fmt.Sprintf(
			"You have been invited to join as staff. Please use the following link to accept the invitation:\n\n%s/%s?email=%s",
			h.staffInvitationBaseURL,
			code,
			url.QueryEscape(email),
		),
	}
	if err := h.mailsender.SendMail(ctx, payload); err != nil {
		return fmt.Errorf("failed to send staff invitation email to %s: %w", logging.RedactEmail(email), err)
	}
	return nil
}
