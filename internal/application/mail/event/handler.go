package mailevent

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/mail"
)

var (
	tracer = otel.Tracer("ucms/application/mail/event")
	logger = otelslog.NewLogger("ucms/application/mail/event")
)

type MailSender interface {
	SendMail(ctx context.Context, payload mail.Payload) error
}

type MailEventHandler struct {
	tracer                 trace.Tracer
	logger                 *slog.Logger
	mailsender             MailSender
	staffInvitationBaseURL string
}

type MailEventHandlerArgs struct {
	Tracer                 trace.Tracer
	Logger                 *slog.Logger
	Mailsender             MailSender
	StaffInvitationBaseURL string
}

func NewMailEventHandler(args MailEventHandlerArgs) *MailEventHandler {
	if args.Tracer == nil {
		args.Tracer = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}

	return &MailEventHandler{
		tracer:                 args.Tracer,
		logger:                 args.Logger,
		mailsender:             args.Mailsender,
		staffInvitationBaseURL: args.StaffInvitationBaseURL,
	}
}
