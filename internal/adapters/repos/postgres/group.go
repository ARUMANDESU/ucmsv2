package postgres

import (
	"context"
	"errors"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/group"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/otelx"
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
		otelx.RecordSpanError(span, err, "failed to execute query")
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorx.NewNotFound().WithCause(err)
		}
		return nil, err
	}

	return GroupToDomain(dto), nil
}

func (r *GroupRepo) SaveGroup(ctx context.Context, g *group.Group) error {
	ctx, span := r.tracer.Start(ctx, "GroupRepo.SaveGroup")
	defer span.End()

	dto := DomainToGroupDTO(g)

	query := `
		INSERT INTO groups (id, name, year, major, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6);
	`

	_, err := r.pool.Exec(ctx, query, dto.ID, dto.Name, dto.Year, dto.Major, dto.CreatedAt, dto.UpdatedAt)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to execute query")
		return err
	}

	return nil
}
