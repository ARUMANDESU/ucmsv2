package mail

import "github.com/ARUMANDESU/ucms/internal/application/mail/event"

type App struct {
	Event Event
}

type Event struct {
	RegistrationStarted *event.RegistrationStartedHandler
}

type Args struct {
	Mailsender event.MailSender
}

func NewApp(args Args) *App {
	return &App{
		Event: Event{
			RegistrationStarted: event.NewRegistrationStartedHandler(event.RegistrationStartedHandlerArgs{
				Mailsender: args.Mailsender,
			}),
		},
	}
}
