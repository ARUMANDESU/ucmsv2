package studentapp

import (
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/application/student/event"
)

type App struct {
	Event Event
}

type Event struct {
	StudentRegistrationCompleted *event.StudentRegistrationCompletedHandler
}

type Args struct {
	StudentRepo event.Repo
	Tracer      trace.Tracer
	Logger      *slog.Logger
}

func NewApp(args Args) *App {
	return &App{
		Event: Event{
			StudentRegistrationCompleted: event.NewStudentRegistrationCompletedHandler(event.StudentRegistrationCompletedHandlerArgs{
				Tracer:      args.Tracer,
				Logger:      args.Logger,
				StudentRepo: args.StudentRepo,
			}),
		},
	}
}
