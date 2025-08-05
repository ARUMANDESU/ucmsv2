package postgres

import (
	"context"
	"errors"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ARUMANDESU/ucms/internal/adapters/repos"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/pkg/postgres"
	"github.com/ARUMANDESU/ucms/pkg/watermillx"
)

type RegistrationRepo struct {
	pool    *pgxpool.Pool
	wlogger watermill.LoggerAdapter
}

func NewRegistrationRepo(pool *pgxpool.Pool) *RegistrationRepo {
	return &RegistrationRepo{
		pool:    pool,
		wlogger: watermill.NewStdLogger(false, false),
	}
}

func (re *RegistrationRepo) GetRegistrationByEmail(ctx context.Context, email string) (*registration.Registration, error) {
	query := `
        SELECT id, email, status, verification_code, code_attempts, code_expires_at, resend_timeout, created_at, updated_at
        FROM registrations
        WHERE email = $1;
    `

	var dto RegistrationDTO
	err := re.pool.QueryRow(ctx, query, email).Scan(
		&dto.ID, &dto.Email, &dto.Status,
		&dto.VerificationCode, &dto.CodeAttempts, &dto.CodeExpiresAt,
		&dto.ResendTimeout, &dto.CreatedAt, &dto.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repos.ErrNotFound
		}
		return nil, err
	}

	return RegistrationToDomain(dto), nil
}

func (re *RegistrationRepo) GetRegistrationByID(ctx context.Context, id registration.ID) (*registration.Registration, error) {
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repos.ErrNotFound
		}
		return nil, err
	}

	return RegistrationToDomain(dto), nil
}

func (re *RegistrationRepo) SaveRegistration(ctx context.Context, r *registration.Registration) error {
	dto := DomainToRegistrationDTO(r)

	query := `
        INSERT INTO registrations (id, email, status, verification_code, code_attempts, code_expires_at, resend_timeout, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `

	return postgres.WithTx(ctx, re.pool, func(ctx context.Context, tx pgx.Tx) error {
		events := r.GetUncommittedEvents()
		res, err := tx.Exec(ctx, query,
			dto.ID, dto.Email, dto.Status,
			dto.VerificationCode, dto.CodeAttempts, dto.CodeExpiresAt,
			dto.ResendTimeout, dto.CreatedAt, dto.UpdatedAt,
		)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return repos.ErrNoRowsAffected
		}

		if len(events) > 0 {
			if err := watermillx.Publish(ctx, tx, re.wlogger, events...); err != nil {
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
			if errors.Is(err, pgx.ErrNoRows) {
				return repos.ErrNotFound
			}
			return err
		}

		reg := RegistrationToDomain(dto)

		if err := fn(ctx, reg); err != nil {
			return err
		}

		dto = DomainToRegistrationDTO(reg)

		res, err := tx.Exec(ctx, updatequery,
			dto.ID, dto.Email, dto.Status,
			dto.VerificationCode, dto.CodeAttempts, dto.CodeExpiresAt,
			dto.ResendTimeout, dto.UpdatedAt,
		)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return repos.ErrNoRowsAffected
		}

		events := reg.GetUncommittedEvents()
		if len(events) > 0 {
			if err := watermillx.Publish(ctx, tx, re.wlogger, events...); err != nil {
				return err
			}
		}
		return nil
	})
}
