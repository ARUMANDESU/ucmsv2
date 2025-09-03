package query

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/pkg/errorx"
)

var (
	tracer = otel.Tracer("ucms/internal/application/registration/query")
	logger = otelslog.NewLogger("ucms/internal/application/registration/query")
)

type GetVerificationCodeHandler struct {
	tracer trace.Tracer
	logger *slog.Logger
	pool   *pgxpool.Pool
}

func NewGetVerificationCodeHandler(pool *pgxpool.Pool) *GetVerificationCodeHandler {
	return &GetVerificationCodeHandler{
		pool:   pool,
		tracer: tracer,
		logger: logger,
	}
}

func (h *GetVerificationCodeHandler) Handle(ctx context.Context, email string) (string, error) {
	const op = "query.GetVerificationCodeHandler.Handle"
	var code string
	err := h.pool.QueryRow(ctx, `
        SELECT verification_code
        FROM registrations
        WHERE email = $1
        ORDER BY created_at DESC
        LIMIT 1
    `, email).Scan(&code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errorx.NewNotFound().WithCause(err, op)
		}
		return "", errorx.Wrap(err, op)
	}
	return code, nil
}
