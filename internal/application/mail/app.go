package mail

import (
	"github.com/ARUMANDESU/ucms/internal/application/mail/event"
)

type App struct {
	Event *event.MailEventHandler
}

type Args struct {
	Mailsender             event.MailSender
	StaffInvitationBaseURL string
}

func NewApp(args Args) *App {
	return &App{
		Event: event.NewMailEventHandler(event.MailEventHandlerArgs{
			Mailsender:             args.Mailsender,
			StaffInvitationBaseURL: args.StaffInvitationBaseURL,
		}),
	}
}
