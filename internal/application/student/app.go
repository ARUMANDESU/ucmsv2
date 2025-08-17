package studentapp

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/application/student/event"
	"github.com/ARUMANDESU/ucms/internal/application/student/studentquery"
)

type App struct {
	Event Event
	Query Query
}

type Event struct {
	StudentRegistrationCompleted *event.StudentRegistrationCompletedHandler
}

type Query struct {
	GetStudent *studentquery.GetStudentHandler
}

type Args struct {
	StudentRepo event.Repo
	PgxPool     *pgxpool.Pool
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
		Query: Query{
			GetStudent: studentquery.NewGetStudentHandler(studentquery.GetStudentHandlerArgs{
				Tracer: args.Tracer,
				Logger: args.Logger,
				Pool:   args.PgxPool,
			}),
		},
	}
}
