package mail

import (
	mailevent "gitlab.com/ucmsv2/ucms-backend/internal/application/mail/event"
)

type App struct {
	Event *mailevent.MailEventHandler
}

type Args struct {
	Mailsender              mailevent.MailSender
	StaffInvitationBaseURL  string
	InvitationCreatorGetter mailevent.InvitationCreatorGetter
}

func NewApp(args Args) *App {
	return &App{
		Event: mailevent.NewMailEventHandler(mailevent.MailEventHandlerArgs{
			Mailsender:              args.Mailsender,
			StaffInvitationBaseURL:  args.StaffInvitationBaseURL,
			InvitationCreatorGetter: args.InvitationCreatorGetter,
		}),
	}
}
