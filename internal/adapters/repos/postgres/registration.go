package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/otelx"
	"github.com/ARUMANDESU/ucms/pkg/postgres"
	"github.com/ARUMANDESU/ucms/pkg/watermillx"
)

type RegistrationRepo struct {
	tracer  trace.Tracer
	logger  *slog.Logger
	pool    *pgxpool.Pool
	wlogger watermill.LoggerAdapter
}

// NewRegistrationRepo creates a new instance of RegistrationRepo.
// It also sets default tracer and logger if they are nil.
//
//	WARNING; panics if pool is nil
func NewRegistrationRepo(pool *pgxpool.Pool, t trace.Tracer, l *slog.Logger) *RegistrationRepo {
	if pool == nil {
		panic("pgxpool.Pool cannot be nil")
	}
	if t == nil {
		t = tracer
	}
	if l == nil {
		l = logger
	}

	return &RegistrationRepo{
		tracer:  t,
		logger:  l,
		pool:    pool,
		wlogger: watermill.NewSlogLogger(l),
	}
}

func (r *RegistrationRepo) GetRegistrationByEmail(ctx context.Context, email string) (*registration.Registration, error) {
	ctx, span := r.tracer.Start(ctx, "RegistrationRepo.GetRegistrationByEmail")
	defer span.End()

	query := `
        SELECT id, email, status, verification_code, code_attempts, code_expires_at, resend_timeout, created_at, updated_at
        FROM registrations
        WHERE email = $1;
    `

	var dto RegistrationDTO
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&dto.ID, &dto.Email, &dto.Status,
		&dto.VerificationCode, &dto.CodeAttempts, &dto.CodeExpiresAt,
		&dto.ResendTimeout, &dto.CreatedAt, &dto.UpdatedAt,
	)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to get registration by email")
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorx.NewNotFound().WithCause(err)
		}
		return nil, err
	}

	return RegistrationToDomain(dto), nil
}

func (re *RegistrationRepo) GetRegistrationByID(ctx context.Context, id registration.ID) (*registration.Registration, error) {
	ctx, span := re.tracer.Start(ctx, "RegistrationRepo.GetRegistrationByID")
	defer span.End()

	query := `
		SELECT id, email, status, verification_code, code_attempts, code_expires_at, resend_timeout, created_at, updated_at
		FROM registrations
		WHERE id = $1;
	`

	var dto RegistrationDTO
	err := re.pool.QueryRow(ctx, query, uuid.UUID(id)).Scan(
		&dto.ID, &dto.Email, &dto.Status,
		&dto.VerificationCode, &dto.CodeAttempts, &dto.CodeExpiresAt,
		&dto.ResendTimeout, &dto.CreatedAt, &dto.UpdatedAt,
	)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to get registration by id")
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorx.NewNotFound().WithCause(err)
		}
		return nil, err
	}

	return RegistrationToDomain(dto), nil
}

func (re *RegistrationRepo) SaveRegistration(ctx context.Context, r *registration.Registration) error {
	ctx, span := re.tracer.Start(ctx, "RegistrationRepo.SaveRegistration")
	defer span.End()

	dto := DomainToRegistrationDTO(r)

	query := `
        INSERT INTO registrations (id, email, status, verification_code, code_attempts, code_expires_at, resend_timeout, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `

	return postgres.WithTx(ctx, re.pool, func(ctx context.Context, tx pgx.Tx) error {
		res, err := tx.Exec(ctx, query,
			dto.ID, dto.Email, dto.Status,
			dto.VerificationCode, dto.CodeAttempts, dto.CodeExpiresAt,
			dto.ResendTimeout, dto.CreatedAt, dto.UpdatedAt,
		)
		if err != nil {
			otelx.RecordSpanError(span, err, "failed to insert registration")
			return err
		}
		if res.RowsAffected() == 0 {
			otelx.RecordSpanError(span, ErrNoRowsAffected, "no rows affected when inserting registration")
			return fmt.Errorf("failed to insert registration: %w", ErrNoRowsAffected)
		}

		if events := r.GetUncommittedEvents(); len(events) > 0 {
			if err := watermillx.Publish(ctx, tx, re.wlogger, events...); err != nil {
				otelx.RecordSpanError(span, err, "failed to publish events")
				return err
			}
		}
		return nil
	})
}

func (re *RegistrationRepo) UpdateRegistration(
	ctx context.Context,
	id registration.ID,
	fn func(ctx context.Context, r *registration.Registration) error,
) error {
	ctx, span := re.tracer.Start(ctx, "RegistrationRepo.UpdateRegistration")
	defer span.End()
	if fn == nil {
		otelx.RecordSpanError(span, ErrNilFunc, "update function cannot be nil")
		return ErrNilFunc
	}

	selectquery := `
        SELECT id, email, status, verification_code, code_attempts, code_expires_at, resend_timeout, created_at, updated_at
        FROM registrations
        WHERE id = $1
        FOR UPDATE;
    `
	updatequery := `
        UPDATE registrations
        SET email = $2, status = $3, verification_code = $4,
            code_attempts = $5, code_expires_at = $6, resend_timeout = $7,
            updated_at = $8
        WHERE id = $1;
    `

	return postgres.WithTx(ctx, re.pool, func(ctx context.Context, tx pgx.Tx) error {
		var dto RegistrationDTO
		err := tx.QueryRow(ctx, selectquery, uuid.UUID(id)).Scan(
			&dto.ID, &dto.Email, &dto.Status,
			&dto.VerificationCode, &dto.CodeAttempts, &dto.CodeExpiresAt,
			&dto.ResendTimeout, &dto.CreatedAt, &dto.UpdatedAt,
		)
		if err != nil {
			otelx.RecordSpanError(span, err, "failed to get registration for update")
			if errors.Is(err, pgx.ErrNoRows) {
				return errorx.NewNotFound().WithCause(err)
			}
			return err
		}

		reg := RegistrationToDomain(dto)

		fnerr := fn(ctx, reg)
		if fnerr != nil && !errorx.IsPersistable(fnerr) {
			otelx.RecordSpanError(span, fnerr, "failed to apply update function")
			return fnerr
		}

		dto = DomainToRegistrationDTO(reg)

		res, err := tx.Exec(ctx, updatequery,
			dto.ID, dto.Email, dto.Status,
			dto.VerificationCode, dto.CodeAttempts, dto.CodeExpiresAt,
			dto.ResendTimeout, dto.UpdatedAt,
		)
		if err != nil {
			otelx.RecordSpanError(span, err, "failed to update registration")
			return err
		}
		if res.RowsAffected() == 0 {
			otelx.RecordSpanError(span, ErrNoRowsAffected, "no rows affected when updating registration")
			return fmt.Errorf("failed to update registration: %w", ErrNoRowsAffected)
		}

		events := reg.GetUncommittedEvents()
		if len(events) > 0 {
			if err := watermillx.Publish(ctx, tx, re.wlogger, events...); err != nil {
				otelx.RecordSpanError(span, err, "failed to publish events")
				return err
			}
		}

		if fnerr != nil && errorx.IsPersistable(fnerr) {
			otelx.RecordSpanError(span, fnerr, "update function returned an error but is allowed to continue")
			return fnerr
		}
		return nil
	})
}

func (re *RegistrationRepo) UpdateRegistrationByEmail(
	ctx context.Context,
	email string,
	fn func(ctx context.Context, r *registration.Registration) error,
) error {
	ctx, span := re.tracer.Start(ctx, "RegistrationRepo.UpdateRegistrationByEmail")
	defer span.End()
	if fn == nil {
		otelx.RecordSpanError(span, ErrNilFunc, "update function cannot be nil")
		return ErrNilFunc
	}

	selectquery := `
        SELECT id, email, status, verification_code, code_attempts, code_expires_at, resend_timeout, created_at, updated_at
        FROM registrations
        WHERE email = $1
        FOR UPDATE;
    `
	updatequery := `
        UPDATE registrations
        SET email = $2, status = $3, verification_code = $4,
            code_attempts = $5, code_expires_at = $6, resend_timeout = $7,
            updated_at = $8
        WHERE id = $1;
    `

	return postgres.WithTx(ctx, re.pool, func(ctx context.Context, tx pgx.Tx) error {
		var dto RegistrationDTO
		err := tx.QueryRow(ctx, selectquery, email).Scan(
			&dto.ID, &dto.Email, &dto.Status,
			&dto.VerificationCode, &dto.CodeAttempts, &dto.CodeExpiresAt,
			&dto.ResendTimeout, &dto.CreatedAt, &dto.UpdatedAt,
		)
		if err != nil {
			otelx.RecordSpanError(span, err, "failed to get registration for update")
			if errors.Is(err, pgx.ErrNoRows) {
				return errorx.NewNotFound().WithCause(err)
			}
			return err
		}

		reg := RegistrationToDomain(dto)

		fnerr := fn(ctx, reg)
		if fnerr != nil && !errorx.IsPersistable(fnerr) {
			otelx.RecordSpanError(span, fnerr, "failed to apply update function")
			return fnerr
		}

		dto = DomainToRegistrationDTO(reg)

		res, err := tx.Exec(ctx, updatequery,
			dto.ID, dto.Email, dto.Status,
			dto.VerificationCode, dto.CodeAttempts, dto.CodeExpiresAt,
			dto.ResendTimeout, dto.UpdatedAt,
		)
		if err != nil {
			otelx.RecordSpanError(span, err, "failed to update registration")
			return err
		}
		if res.RowsAffected() == 0 {
			return fmt.Errorf("failed to update registration: %w", ErrNoRowsAffected)
		}

		events := reg.GetUncommittedEvents()
		if len(events) > 0 {
			if err := watermillx.Publish(ctx, tx, re.wlogger, events...); err != nil {
				otelx.RecordSpanError(span, err, "failed to publish events")
				return err
			}
		}

		if fnerr != nil && errorx.IsPersistable(fnerr) {
			otelx.RecordSpanError(span, fnerr, "update function returned an error but is allowed to continue")
			return fnerr
		}
		return nil
	})
}
