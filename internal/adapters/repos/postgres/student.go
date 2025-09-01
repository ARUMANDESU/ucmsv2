package postgres

import (
	"context"
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

type StudentRepo struct {
	tracer  trace.Tracer
	logger  *slog.Logger
	pool    *pgxpool.Pool
	wlogger watermill.LoggerAdapter
}

func NewStudentRepo(pool *pgxpool.Pool, t trace.Tracer, l *slog.Logger) *StudentRepo {
	if pool == nil {
		panic("pgxpool.Pool cannot be nil")
	}
	if t == nil {
		t = tracer
	}
	if l == nil {
		l = logger
	}

	return &StudentRepo{
		tracer:  t,
		logger:  l,
		pool:    pool,
		wlogger: watermill.NewSlogLogger(l),
	}
}

func (st *StudentRepo) SaveStudent(ctx context.Context, student *user.Student) error {
	ctx, span := st.tracer.Start(ctx, "StudentRepo.SaveStudent")
	defer span.End()

	return postgres.WithTx(ctx, st.pool, func(ctx context.Context, tx pgx.Tx) error {
		dto := DomainToUserDTO(student.User(), 0)
		res, err := tx.Exec(ctx, insertUserQuery,
			dto.ID,
			dto.Barcode,
			dto.Username,
			student.User().Role().String(),
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

		insertStudentQuery := `
            INSERT INTO students (user_id, group_id, created_at, updated_at)
            VALUES ($1, $2, $3, $4);
        `
		res, err = tx.Exec(ctx, insertStudentQuery,
			dto.ID,
			student.GroupID(),
			dto.CreatedAt,
			dto.UpdatedAt,
		)
		if err != nil {
			otelx.RecordSpanError(span, err, "failed to insert student")
			return err
		}
		if res.RowsAffected() == 0 {
			err := fmt.Errorf("no rows affected while inserting student: %w", ErrNoRowsAffected)
			otelx.RecordSpanError(span, err, "no rows affected while inserting student")
			return err
		}

		events := student.GetUncommittedEvents()
		if len(events) > 0 {
			if err := watermillx.Publish(ctx, tx, st.wlogger, events...); err != nil {
				otelx.RecordSpanError(span, err, "failed to publish events")
				return err
			}
		}
		return nil
	})
}
