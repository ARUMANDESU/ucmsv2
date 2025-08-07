package registration

import (
	"github.com/ARUMANDESU/ucms/internal/application/registration/cmd"
	"github.com/ARUMANDESU/ucms/pkg/env"
)

type App struct {
	CMD   Command
	Event Event
}

type Command struct {
	StartStudent *cmd.StartStudentHandler
}

type Event struct{}

type Args struct {
	Mode       env.Mode
	Repo       cmd.Repo
	UserGetter cmd.UserGetter
}

func NewApp(args Args) *App {
	return &App{
		CMD: Command{
			StartStudent: cmd.NewStartStudentHandler(cmd.StartStudentHandlerArgs{
				Mode:       args.Mode,
				Repo:       args.Repo,
				UserGetter: args.UserGetter,
			}),
		},
	}
}
