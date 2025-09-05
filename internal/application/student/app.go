package studentapp

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"

	"gitlab.com/ucmsv2/ucms-backend/internal/application/student/studentquery"
)

type App struct {
	Event Event
	Query Query
}

type Event struct{}

type Query struct {
	GetStudent *studentquery.GetStudentHandler
}

type Args struct {
	PgxPool *pgxpool.Pool
	Tracer  trace.Tracer
	Logger  *slog.Logger
}

func NewApp(args Args) *App {
	return &App{
		Event: Event{},
		Query: Query{
			GetStudent: studentquery.NewGetStudentHandler(studentquery.GetStudentHandlerArgs{
				Tracer: args.Tracer,
				Logger: args.Logger,
				Pool:   args.PgxPool,
			}),
		},
	}
}
