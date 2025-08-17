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
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/pkg/errorx"
)

var (
	tracer = otel.Tracer("ucms/internal/application/student/query")
	logger = otelslog.NewLogger("ucms/internal/application/student/query")
)

type GetStudent struct {
	ID string `json:"id"`
}

type GetStudentResponse struct {
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
	ctx, span := h.tracer.Start(ctx, "GetStudentHandler.Handle",
		trace.WithAttributes(attribute.String("student.id", query.ID)),
	)
	defer span.End()

	var res GetStudentResponse
	err := h.pool.QueryRow(ctx, `
        SELECT u.id, u.email, u.first_name, u.last_name, u.avatar_url, u.created_at, 
            gr.name, g.id, g.major, g.name, g.year
        FROM students s JOIN users u ON s.user_id = u.id
        JOIN groups g ON s.group_id = g.id
        JOIN global_roles gr ON u.role_id = gr.id
        WHERE u.id = $1
    `, query.ID).Scan(
		&res.Barcode, &res.Email, &res.FirstName, &res.LastName, &res.AvatarURL,
		&res.RegisteredAt, &res.Role, &res.Group.ID, &res.Group.Major, &res.Group.Name, &res.Group.Year,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query student data")
		if errors.Is(err, pgx.ErrNoRows) {
			h.logger.Warn("student not found", "id", query.ID)
			return nil, errorx.NewNotFound().WithCause(err)
		}
		return nil, err
	}

	return &res, nil
}
