package mail

import (
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/application/mail/event"
)

type App struct {
	Event Event
}

type Event struct {
	RegistrationStarted *event.RegistrationStartedHandler
}

type Args struct {
	Mailsender event.MailSender
	Tracer     trace.Tracer
	Logger     *slog.Logger
}

func NewApp(args Args) *App {
	return &App{
		Event: Event{
			RegistrationStarted: event.NewRegistrationStartedHandler(event.RegistrationStartedHandlerArgs{
				Mailsender: args.Mailsender,
				Tracer:     args.Tracer,
				Logger:     args.Logger,
			}),
		},
	}
}
