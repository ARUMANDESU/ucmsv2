package studentquery

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/otelx"
)

var (
	tracer = otel.Tracer("ucms/internal/application/student/query")
	logger = otelslog.NewLogger("ucms/internal/application/student/query")
)

type GetStudent struct {
	ID user.ID `json:"id"`
}

type GetStudentResponse struct {
	ID        string `json:"id"`
	Barcode   string `json:"barcode"`
	GroupID   string `json:"group_id"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
	Group     struct {
		ID    string `json:"id"`
		Major string `json:"major"`
		Name  string `json:"name"`
		Year  string `json:"year"`
	} `json:"group"`
	RegisteredAt time.Time `json:"registered_at"`
}

type GetStudentHandler struct {
	tracer trace.Tracer
	logger *slog.Logger
	pool   *pgxpool.Pool
}

type GetStudentHandlerArgs struct {
	Tracer trace.Tracer
	Logger *slog.Logger
	Pool   *pgxpool.Pool
}

func NewGetStudentHandler(args GetStudentHandlerArgs) *GetStudentHandler {
	if args.Tracer == nil {
		args.Tracer = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}

	return &GetStudentHandler{
		tracer: args.Tracer,
		logger: args.Logger,
		pool:   args.Pool,
	}
}

func (h *GetStudentHandler) Handle(ctx context.Context, query GetStudent) (*GetStudentResponse, error) {
	const op = "studentquery.GetStudentHandler.Handle"
	ctx, span := h.tracer.Start(ctx, "GetStudentHandler.Handle",
		trace.WithAttributes(attribute.String("student.id", query.ID.String())),
	)
	defer span.End()

	var res GetStudentResponse
	err := h.pool.QueryRow(ctx, `
        SELECT u.id, u.barcode, u.email, u.first_name, u.last_name, u.avatar_url, u.created_at, 
            gr.name, g.id, g.major, g.name, g.year
        FROM students s JOIN users u ON s.user_id = u.id
        JOIN groups g ON s.group_id = g.id
        JOIN global_roles gr ON u.role_id = gr.id
        WHERE u.id = $1
    `, query.ID).Scan(
		&res.ID, &res.Barcode, &res.Email, &res.FirstName, &res.LastName, &res.AvatarURL,
		&res.RegisteredAt, &res.Role, &res.Group.ID, &res.Group.Major, &res.Group.Name, &res.Group.Year,
	)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to get student by id")
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorx.NewNotFound().WithCause(err, op)
		}
		return nil, errorx.Wrap(err, op)
	}

	return &res, nil
}
