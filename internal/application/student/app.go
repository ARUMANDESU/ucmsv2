package studentapp

import (
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
}

func NewApp(args Args) *App {
	return &App{
		Event: Event{
			StudentRegistrationCompleted: event.NewStudentRegistrationCompletedHandler(event.StudentRegistrationCompletedHandlerArgs{
				StudentRepo: args.StudentRepo,
			}),
		},
	}
}
