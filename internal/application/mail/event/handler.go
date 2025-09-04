package mailevent

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/staffinvitation"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/mail"
)

var (
	tracer = otel.Tracer("ucms/application/mail/event")
	logger = otelslog.NewLogger("ucms/application/mail/event")
)

type InvitationCreatorGetter interface {
	GetCreatorByInvitationID(ctx context.Context, id staffinvitation.ID) (*user.Staff, error)
}

type MailSender interface {
	SendMail(ctx context.Context, payload mail.Payload) error
}

type MailEventHandler struct {
	tracer                  trace.Tracer
	logger                  *slog.Logger
	mailsender              MailSender
	staffInvitationBaseURL  string
	invitationCreatorGetter InvitationCreatorGetter
}

type MailEventHandlerArgs struct {
	Tracer                  trace.Tracer
	Logger                  *slog.Logger
	StaffInvitationBaseURL  string
	Mailsender              MailSender
	InvitationCreatorGetter InvitationCreatorGetter
}

func NewMailEventHandler(args MailEventHandlerArgs) *MailEventHandler {
	if args.Tracer == nil {
		args.Tracer = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}

	return &MailEventHandler{
		tracer:                  args.Tracer,
		logger:                  args.Logger,
		staffInvitationBaseURL:  args.StaffInvitationBaseURL,
		mailsender:              args.Mailsender,
		invitationCreatorGetter: args.InvitationCreatorGetter,
	}
}
