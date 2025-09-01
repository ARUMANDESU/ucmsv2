package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/otelx"
	"github.com/ARUMANDESU/ucms/pkg/postgres"
	"github.com/ARUMANDESU/ucms/pkg/watermillx"
)

type StaffRepo struct {
	tracer  trace.Tracer
	logger  *slog.Logger
	pool    *pgxpool.Pool
	wlogger watermill.LoggerAdapter
}

func NewStaffRepo(pool *pgxpool.Pool, t trace.Tracer, l *slog.Logger) *StaffRepo {
	if pool == nil {
		panic("pgxpool.Pool cannot be nil")
	}
	if t == nil {
		t = tracer
	}
	if l == nil {
		l = logger
	}

	return &StaffRepo{
		tracer:  t,
		logger:  l,
		pool:    pool,
		wlogger: watermill.NewSlogLogger(l),
	}
}

func (r *StaffRepo) HasAnyStaff(ctx context.Context) (bool, error) {
	ctx, span := r.tracer.Start(ctx, "StaffRepo.HasAnyStaff")
	defer span.End()

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM staffs);`
	err := r.pool.QueryRow(ctx, query).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		otelx.RecordSpanError(span, err, "failed to check if any staff exists")
		return false, err
	}
	return exists, nil
}

func (r *StaffRepo) SaveStaff(ctx context.Context, staff *user.Staff) error {
	ctx, span := r.tracer.Start(ctx, "StaffRepo.SaveStaff")
	defer span.End()

	return postgres.WithTx(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		dto := DomainToUserDTO(staff.User(), 0)
		res, err := tx.Exec(ctx, insertUserQuery,
			dto.ID,
			dto.Barcode,
			dto.Username,
			staff.User().Role().String(),
			dto.Email,
			dto.FirstName,
			dto.LastName,
			dto.AvatarURL,
			dto.Passhash,
			dto.CreatedAt,
			dto.UpdatedAt,
		)
		if err != nil {
			otelx.RecordSpanError(span, err, "failed to insert user")
			return err
		}
		if res.RowsAffected() == 0 {
			err := fmt.Errorf("no rows affected while inserting user: %w", ErrNoRowsAffected)
			otelx.RecordSpanError(span, err, "no rows affected while inserting user")
			return err
		}

		insertStaffQuery := `
            INSERT INTO staffs (user_id)
            VALUES ($1);
        `
		res, err = tx.Exec(ctx, insertStaffQuery, dto.ID)
		if err != nil {
			otelx.RecordSpanError(span, err, "failed to insert staff")
			return err
		}
		if res.RowsAffected() == 0 {
			err := fmt.Errorf("no rows affected while inserting staff: %w", ErrNoRowsAffected)
			otelx.RecordSpanError(span, err, "no rows affected while inserting staff")
			return err
		}

		events := staff.GetUncommittedEvents()
		if len(events) > 0 {
			if err := watermillx.Publish(ctx, tx, r.wlogger, events...); err != nil {
				otelx.RecordSpanError(span, err, "failed to publish events")
				return err
			}
		}
		return nil
	})
}
