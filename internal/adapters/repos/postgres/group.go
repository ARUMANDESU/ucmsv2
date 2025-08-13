package postgres

import (
	"context"
	"errors"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/group"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
)

type GroupRepo struct {
	tracer  trace.Tracer
	logger  *slog.Logger
	pool    *pgxpool.Pool
	wlogger watermill.LoggerAdapter
}

// NewGroupRepo creates a new instance of GroupRepo.
// It also sets default tracer and logger if they are nil.
//
//	WARNING: panics if pool is nil
func NewGroupRepo(pool *pgxpool.Pool, t trace.Tracer, l *slog.Logger) *GroupRepo {
	if pool == nil {
		panic("pgxpool.Pool cannot be nil")
	}
	if t == nil {
		t = tracer
	}
	if l == nil {
		l = logger
	}

	return &GroupRepo{
		tracer:  t,
		logger:  l,
		pool:    pool,
		wlogger: watermill.NewSlogLogger(l),
	}
}

func (r *GroupRepo) GetGroupByID(ctx context.Context, groupID group.ID) (*group.Group, error) {
	ctx, span := r.tracer.Start(ctx, "GroupRepo.GetGroupByID")
	defer span.End()

	query := `
        SELECT id, name, year, major, created_at, updated_at
        FROM groups
        WHERE id = $1;
    `

	var dto GroupDTO
	err := r.pool.QueryRow(ctx, query, groupID).Scan(
		&dto.ID,
		&dto.Name,
		&dto.Year,
		&dto.Major,
		&dto.CreatedAt,
		&dto.UpdatedAt,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get group by ID")
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorx.NewNotFound().WithCause(err)
		}
		return nil, err
	}

	return GroupToDomain(dto), nil
}
