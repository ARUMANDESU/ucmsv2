package mail

import (
	mailevent "github.com/ARUMANDESU/ucms/internal/application/mail/event"
)

type App struct {
	Event *mailevent.MailEventHandler
}

type Args struct {
	Mailsender             mailevent.MailSender
	StaffInvitationBaseURL string
}

func NewApp(args Args) *App {
	return &App{
		Event: mailevent.NewMailEventHandler(mailevent.MailEventHandlerArgs{
			Mailsender:             args.Mailsender,
			StaffInvitationBaseURL: args.StaffInvitationBaseURL,
		}),
	}
}
