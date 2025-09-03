package postgres

import (
	"context"
	"errors"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/staffinvitation"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/otelx"
	"github.com/ARUMANDESU/ucms/pkg/postgres"
	"github.com/ARUMANDESU/ucms/pkg/watermillx"
)

type StaffInvitationRepo struct {
	tracer  trace.Tracer
	pool    *pgxpool.Pool
	wlogger watermill.LoggerAdapter
}

// NewStaffInvitationRepo creates a new StaffInvitationRepo.
// It also sets default tracer and logger if they are nil.
//
//	WARNING; panics if pool is nil
func NewStaffInvitationRepo(pool *pgxpool.Pool, t trace.Tracer, l *slog.Logger) *StaffInvitationRepo {
	if pool == nil {
		panic("pgxpool.Pool is required")
	}
	if t == nil {
		t = tracer
	}
	if l == nil {
		l = logger
	}

	return &StaffInvitationRepo{
		tracer:  t,
		pool:    pool,
		wlogger: watermill.NewSlogLogger(l),
	}
}

func (r *StaffInvitationRepo) SaveStaffInvitation(ctx context.Context, invitation *staffinvitation.StaffInvitation) error {
	ctx, span := r.tracer.Start(ctx, "StaffInvitationRepo.SaveStaffInvitation")
	defer span.End()

	dto := DomainToStaffInvitationDTO(invitation)

	query := `
        INSERT INTO staff_invitations (id, creator_id, code, recipients_email, valid_from, valid_until, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `

	err := postgres.WithTx(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		res, err := r.pool.Exec(ctx, query,
			dto.ID,
			dto.CreatorID,
			dto.Code,
			dto.RecipientsEmail,
			dto.ValidFrom,
			dto.ValidUntil,
			dto.CreatedAt,
			dto.UpdatedAt,
		)
		if err != nil {
			otelx.RecordSpanError(span, err, "failed to execute insert query")
			return err
		}
		if res.RowsAffected() == 0 {
			otelx.RecordSpanError(span, ErrNoRowsAffected, "no rows affected when inserting staff invitation")
			return ErrNoRowsAffected
		}

		if events := invitation.GetUncommittedEvents(); len(events) > 0 {
			if err := watermillx.Publish(ctx, tx, r.wlogger, events...); err != nil {
				otelx.RecordSpanError(span, err, "failed to publish events")
				return err
			}
		}

		return nil
	})
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to execute transaction")
		return err
	}

	return nil
}

func (r *StaffInvitationRepo) UpdateStaffInvitation(
	ctx context.Context,
	id staffinvitation.ID,
	fn func(context.Context, *staffinvitation.StaffInvitation) error,
) error {
	ctx, span := r.tracer.Start(ctx, "StaffInvitationRepo.UpdateStaffInvitation")
	defer span.End()
	if fn == nil {
		otelx.RecordSpanError(span, ErrNilFunc, "update function cannot be nil")
		return ErrNilFunc
	}

	selectquery := `
        SELECT id, creator_id, code, recipients_email, valid_from, valid_until, created_at, updated_at, deleted_at
        FROM staff_invitations
        WHERE id = $1
        FOR UPDATE;
    `
	updatequery := `
        UPDATE staff_invitations
        SET creator_id = $2, code = $3, recipients_email = $4, valid_from = $5,
            valid_until = $6, updated_at = $7, deleted_at = $8
        WHERE id = $1;
    `
	err := postgres.WithTx(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		var dto StaffInvitationDTO
		err := tx.QueryRow(ctx, selectquery, id).Scan(
			&dto.ID, &dto.CreatorID, &dto.Code, &dto.RecipientsEmail,
			&dto.ValidFrom, &dto.ValidUntil, &dto.CreatedAt,
			&dto.UpdatedAt, &dto.DeletedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errorx.NewNotFound().WithCause(err)
			}
			otelx.RecordSpanError(span, err, "failed to select staff invitation")
			return err
		}

		invitation := StaffInvitationToDomain(dto)

		if err := fn(ctx, invitation); err != nil {
			otelx.RecordSpanError(span, err, "update function failed")
			return err
		}

		dto = DomainToStaffInvitationDTO(invitation)
		res, err := tx.Exec(ctx, updatequery,
			dto.ID,
			dto.CreatorID,
			dto.Code,
			dto.RecipientsEmail,
			dto.ValidFrom,
			dto.ValidUntil,
			dto.UpdatedAt,
			dto.DeletedAt,
		)
		if err != nil {
			otelx.RecordSpanError(span, err, "failed to execute update query")
			return err
		}
		if res.RowsAffected() == 0 {
			otelx.RecordSpanError(span, ErrNoRowsAffected, "no rows affected when updating staff invitation")
			return ErrNoRowsAffected
		}

		if events := invitation.GetUncommittedEvents(); len(events) > 0 {
			if err := watermillx.Publish(ctx, tx, r.wlogger, events...); err != nil {
				otelx.RecordSpanError(span, err, "failed to publish events")
				return err
			}
		}

		return nil
	})
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to execute transaction")
		return err
	}

	return nil
}

func (r *StaffInvitationRepo) GetStaffInvitationByID(ctx context.Context, id staffinvitation.ID) (*staffinvitation.StaffInvitation, error) {
	ctx, span := r.tracer.Start(ctx, "StaffInvitationRepo.GetStaffInvitationByID")
	defer span.End()

	query := `
        SELECT id, creator_id, code, recipients_email, valid_from, valid_until, created_at, updated_at, deleted_at
        FROM staff_invitations
        WHERE id = $1;
    `

	var dto StaffInvitationDTO
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&dto.ID, &dto.CreatorID, &dto.Code,
		&dto.RecipientsEmail, &dto.ValidFrom, &dto.ValidUntil,
		&dto.CreatedAt, &dto.UpdatedAt, &dto.DeletedAt,
	)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to execute select query")
		if err == pgx.ErrNoRows {
			return nil, errorx.NewNotFound().WithCause(err)
		}
		return nil, err
	}

	invitation := StaffInvitationToDomain(dto)
	return invitation, nil
}

func (r *StaffInvitationRepo) GetStaffInvitationByCode(ctx context.Context, code string) (*staffinvitation.StaffInvitation, error) {
	ctx, span := r.tracer.Start(ctx, "StaffInvitationRepo.GetStaffInvitationByCode")
	defer span.End()

	query := `
        SELECT id, creator_id, code, recipients_email, valid_from, valid_until, created_at, updated_at, deleted_at
        FROM staff_invitations
        WHERE code = $1;
    `

	var dto StaffInvitationDTO
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&dto.ID, &dto.CreatorID, &dto.Code,
		&dto.RecipientsEmail, &dto.ValidFrom, &dto.ValidUntil,
		&dto.CreatedAt, &dto.UpdatedAt, &dto.DeletedAt,
	)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to execute select query")
		if err == pgx.ErrNoRows {
			return nil, errorx.NewNotFound().WithCause(err)
		}
		return nil, err
	}

	invitation := StaffInvitationToDomain(dto)
	return invitation, nil
}

func (r *StaffInvitationRepo) GetLatestStaffInvitationByCreatorID(
	ctx context.Context,
	creatorID user.ID,
) (*staffinvitation.StaffInvitation, error) {
	ctx, span := r.tracer.Start(ctx, "StaffInvitationRepo.GetLatestStaffInvitationByCreatorID")
	defer span.End()

	query := `
        SELECT id, creator_id, code, recipients_email, valid_from, valid_until, created_at, updated_at, deleted_at
        FROM staff_invitations
        WHERE creator_id = $1
        ORDER BY created_at DESC
        LIMIT 1;
    `

	var dto StaffInvitationDTO
	err := r.pool.QueryRow(ctx, query, creatorID).Scan(
		&dto.ID, &dto.CreatorID, &dto.Code,
		&dto.RecipientsEmail, &dto.ValidFrom, &dto.ValidUntil,
		&dto.CreatedAt, &dto.UpdatedAt, &dto.DeletedAt,
	)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to execute select query")
		if err == pgx.ErrNoRows {
			return nil, errorx.NewNotFound().WithCause(err)
		}
		return nil, err
	}

	invitation := StaffInvitationToDomain(dto)
	return invitation, nil
}
