package registration

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"gitlab.com/ucmsv2/ucms-backend/internal/application/registration/cmd"
	"gitlab.com/ucmsv2/ucms-backend/internal/application/registration/event"
	"gitlab.com/ucmsv2/ucms-backend/internal/application/registration/query"
	"gitlab.com/ucmsv2/ucms-backend/pkg/env"
)

type App struct {
	Command Command
	Event   Event
	Query   Query
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

type Query struct {
	// GetVerificationCode is query handler that returns verification code for email.
	// 	This is only for dev and local environments.
	GetVerificationCode *query.GetVerificationCodeHandler
}

type Args struct {
	Mode         env.Mode
	Repo         cmd.Repo
	UserGetter   cmd.UserGetter
	GroupGetter  cmd.GroupGetter
	StudentSaver cmd.StudentSaver
	PgxPool      *pgxpool.Pool
}

func NewApp(args Args) *App {
	return &App{
		Command: Command{
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
		Query: Query{
			GetVerificationCode: query.NewGetVerificationCodeHandler(args.PgxPool),
		},
	}
}
