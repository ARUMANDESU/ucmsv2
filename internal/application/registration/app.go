package registration

import (
	"github.com/ARUMANDESU/ucms/internal/application/registration/cmd"
	"github.com/ARUMANDESU/ucms/internal/application/registration/event"
	"github.com/ARUMANDESU/ucms/pkg/env"
)

type App struct {
	CMD   Command
	Event Event
}

type Command struct {
	Verify          *cmd.VerifyHandler
	StartStudent    *cmd.StartStudentHandler
	StudentComplete *cmd.StudentCompleteHandler
	ResendCode      *cmd.ResendCodeHandler
}

type Event struct {
	Registration *event.RegistrationCompletedHandler
}

type Args struct {
	Mode         env.Mode
	Repo         cmd.Repo
	UserGetter   cmd.UserGetter
	GroupGetter  cmd.GroupGetter
	StudentSaver cmd.StudentSaver
}

func NewApp(args Args) *App {
	return &App{
		CMD: Command{
			StartStudent: cmd.NewStartStudentHandler(cmd.StartStudentHandlerArgs{
				Mode:       args.Mode,
				Repo:       args.Repo,
				UserGetter: args.UserGetter,
			}),
			Verify: cmd.NewVerifyHandler(cmd.VerifyHandlerArgs{
				RegistrationRepo: args.Repo,
			}),
			StudentComplete: cmd.NewStudentCompleteHandler(cmd.StudentCompleteHandlerArgs{
				UserGetter:       args.UserGetter,
				RegistrationRepo: args.Repo,
				GroupGetter:      args.GroupGetter,
				StudentSaver:     args.StudentSaver,
			}),
			ResendCode: cmd.NewResendCodeHandler(cmd.ResendCodeHandlerArgs{
				Repo:       args.Repo,
				UserGetter: args.UserGetter,
			}),
		},
		Event: Event{
			Registration: event.NewRegistrationCompletedHandler(event.RegistrationCompletedHandlerArgs{
				RegRepo: args.Repo,
			}),
		},
	}
}
