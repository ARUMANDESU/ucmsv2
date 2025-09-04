package mailevent

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/staffinvitation"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/mail"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/logging"
	"github.com/ARUMANDESU/ucms/pkg/otelx"
)

const (
	StaffInvitationSubject = "Staff Invitation"
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

// HandleStaffInvitationAccepted handles the event when a staff invitation is accepted.
//
// Sends Welcome email to the new staff member and notify admin (if needed).
func (h *MailEventHandler) HandleStaffInvitationAccepted(ctx context.Context, e *user.StaffInvitationAccepted) error {
	if e == nil {
		return nil
	}
	const op = "event.MailEventHandler.HandleStaffInvitationAccepted"
	ctx, span := h.tracer.Start(ctx, "MailEventHandler.HandleStaffInvitationAccepted",
		trace.WithNewRoot(),
		trace.WithLinks(trace.LinkFromContext(e.Extract())),
		trace.WithAttributes(
			attribute.String("staff.id", e.StaffID.String()),
			attribute.String("staff.email", logging.RedactEmail(e.Email)),
			attribute.String("invitation.id", e.InvitationID.String()),
		),
	)
	defer span.End()
	l := h.logger.With(
		slog.String("event", "StaffInvitationAccepted"),
		slog.String("staff.id", e.StaffID.String()),
		slog.String("staff.email", logging.RedactEmail(e.Email)),
		slog.String("invitation.id", e.InvitationID.String()),
	)

	newStaffWelcomePayload := mail.Payload{
		To:      e.Email,
		Subject: "Welcome to the Staff Team",
		Body: fmt.Sprintf(
			"Hello,\n\nWelcome to the staff team! Your account has been successfully created.\n\nYou can log in using your email: %s\n\nBest regards,\nThe Team",
			e.Email,
		),
	}

	if err := h.mailsender.SendMail(ctx, newStaffWelcomePayload); err != nil {
		otelx.RecordSpanError(span, err, "failed to send welcome email to new staff")
		l.ErrorContext(ctx, "failed to send welcome email to new staff",
			slog.String("error", err.Error()),
		)
		return errorx.Wrap(err, op)
	}

	creator, err := h.invitationCreatorGetter.GetCreatorByInvitationID(ctx, staffinvitation.ID(e.InvitationID))
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to get invitation creator")
		l.ErrorContext(ctx, "failed to get invitation creator",
			slog.String("error", err.Error()),
		)
		return nil // Do not return error to avoid blocking staff creation process
	}

	notificationPayload := mail.Payload{
		To:      creator.User().Email(),
		Subject: "Staff Invitation Accepted",
		Body: fmt.Sprintf(
			"Hello,\n\nThe staff invitation you sent has been accepted by %s %s (%s).\n\nBest regards,\nThe Team",
			e.FirstName,
			e.LastName,
			e.Email,
		),
	}
	if err := h.mailsender.SendMail(ctx, notificationPayload); err != nil {
		otelx.RecordSpanError(span, err, "failed to send staff invitation accepted notification to creator")
		l.ErrorContext(ctx, "failed to send staff invitation accepted notification to creator",
			slog.String("error", err.Error()),
		)
		// Do not return error to avoid blocking staff creation process
	}

	return nil
}

func (h *MailEventHandler) sendStaffInvitationEmail(ctx context.Context, email, code string) error {
	const op = "mailevent.sendStaffInvitationEmail"
	payload := mail.Payload{
		To:      email,
		Subject: StaffInvitationSubject,
		Body: fmt.Sprintf(
			"You have been invited to join as staff. Please use the following link to accept the invitation:\n\n%s/%s?email=%s",
			h.staffInvitationBaseURL,
			code,
			url.QueryEscape(email),
		),
	}
	if err := h.mailsender.SendMail(ctx, payload); err != nil {
		return errorx.Wrap(err, op)
	}
	return nil
}
