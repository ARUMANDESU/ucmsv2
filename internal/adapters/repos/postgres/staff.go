package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/staffinvitation"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/pkg/env"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/logging"
	"gitlab.com/ucmsv2/ucms-backend/pkg/otelx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/postgres"
	"gitlab.com/ucmsv2/ucms-backend/pkg/watermillx"
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
		wlogger: watermillx.NewOTelFilteredSlogLogger(l, env.Current().SlogLevel()),
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
		dto := DomainToUserDTO(staff.User())
		res, err := tx.Exec(ctx, insertUserQuery,
			dto.ID,
			dto.Barcode,
			dto.Username,
			staff.User().Role().String(),
			dto.Email,
			dto.FirstName,
			dto.LastName,
			dto.AvatarSource,
			dto.AvatarExternal,
			dto.AvatarS3Key,
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

func (r *StaffRepo) GetStaffByID(ctx context.Context, id user.ID) (*user.Staff, error) {
	const op = "postgres.StaffRepo.GetStaffByID"
	ctx, span := r.tracer.Start(ctx, "StaffRepo.GetStaffByID",
		trace.WithAttributes(attribute.String("user.id", id.String())),
	)
	defer span.End()

	query := `
        SELECT  s.user_id, u.id, u.barcode, u.username, 
				u.role_id, u.first_name, u.last_name, 
				u.avatar_source, u.avatar_external, u.avatar_s3_key,
                u.email, u.pass_hash, u.created_at, u.updated_at,
                gr.id, gr.name
        FROM staffs s
        JOIN users u ON s.user_id = u.id
        JOIN global_roles gr ON u.role_id = gr.id
        WHERE s.user_id = $1;
    `

	var userDTO UserDTO
	var roleDTO GlobalRoleDTO
	var staffDTO StaffDTO
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&staffDTO.ID, &userDTO.ID, &userDTO.Barcode, &userDTO.Username,
		&userDTO.RoleID, &userDTO.FirstName, &userDTO.LastName,
		&userDTO.AvatarSource, &userDTO.AvatarExternal, &userDTO.AvatarS3Key,
		&userDTO.Email, &userDTO.Passhash, &userDTO.CreatedAt, &userDTO.UpdatedAt,
		&roleDTO.ID, &roleDTO.Name,
	)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to get staff by id")
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorx.NewNotFound().WithCause(err, op)
		}
		return nil, errorx.Wrap(err, op)
	}

	return StaffToDomain(userDTO, roleDTO, staffDTO), nil
}

func (r *StaffRepo) GetStaffByEmail(ctx context.Context, email string) (*user.Staff, error) {
	const op = "postgres.StaffRepo.GetStaffByEmail"
	ctx, span := r.tracer.Start(ctx, "StaffRepo.GetStaffByEmail",
		trace.WithAttributes(attribute.String("user.email", logging.RedactEmail(email))),
	)
	defer span.End()

	query := `
        SELECT 	s.user_id, u.id, u.barcode, u.username, 
				u.role_id, u.first_name, u.last_name, 
				u.avatar_source, u.avatar_external, u.avatar_s3_key,
                u.email, u.pass_hash, u.created_at, u.updated_at,
                gr.id, gr.name
        FROM staffs s
        JOIN users u ON s.user_id = u.id
        JOIN global_roles gr ON u.role_id = gr.id
        WHERE u.email = $1;
    `

	var userDTO UserDTO
	var roleDTO GlobalRoleDTO
	var staffDTO StaffDTO
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&staffDTO.ID, &userDTO.ID, &userDTO.Barcode, &userDTO.Username,
		&userDTO.RoleID, &userDTO.FirstName, &userDTO.LastName,
		&userDTO.AvatarSource, &userDTO.AvatarExternal, &userDTO.AvatarS3Key,
		&userDTO.Email, &userDTO.Passhash, &userDTO.CreatedAt, &userDTO.UpdatedAt,
		&roleDTO.ID, &roleDTO.Name,
	)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to get staff by email")
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorx.NewNotFound().WithCause(err, op)
		}
		return nil, errorx.Wrap(err, op)
	}

	return StaffToDomain(userDTO, roleDTO, staffDTO), nil
}

func (r *StaffRepo) GetCreatorByInvitationID(ctx context.Context, id staffinvitation.ID) (*user.Staff, error) {
	const op = "postgres.StaffRepo.GetCreatorByInvitationID"
	ctx, span := r.tracer.Start(ctx, "StaffRepo.GetCreatorByInvitationID",
		trace.WithAttributes(attribute.String("invitation.id", id.String())),
	)
	defer span.End()

	query := `
        SELECT s.user_id, u.id, u.barcode, u.username, 
				u.role_id, u.first_name, u.last_name, 
				u.avatar_source, u.avatar_external, u.avatar_s3_key,
                u.email, u.pass_hash, u.created_at, u.updated_at,
                gr.id, gr.name
        FROM staff_invitations si
        JOIN staffs s ON si.creator_id = s.user_id
        JOIN users u ON s.user_id = u.id
        JOIN global_roles gr ON u.role_id = gr.id
        WHERE si.id = $1;
    `

	var userDTO UserDTO
	var roleDTO GlobalRoleDTO
	var staffDTO StaffDTO
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&staffDTO.ID, &userDTO.ID, &userDTO.Barcode, &userDTO.Username,
		&userDTO.RoleID, &userDTO.FirstName, &userDTO.LastName,
		&userDTO.AvatarSource, &userDTO.AvatarExternal, &userDTO.AvatarS3Key,
		&userDTO.Email, &userDTO.Passhash, &userDTO.CreatedAt, &userDTO.UpdatedAt,
		&roleDTO.ID, &roleDTO.Name,
	)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to get creator by invitation id")
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorx.NewNotFound().WithCause(err, op)
		}
		return nil, errorx.Wrap(err, op)
	}

	return StaffToDomain(userDTO, roleDTO, staffDTO), nil
}

func (st *StaffRepo) IsStaffExists(
	ctx context.Context,
	email string,
	username string,
	barcode user.Barcode,
) (emailExists bool, usernameExists bool, barcodeExists bool, err error) {
	const op = "postgres.StaffRepo.IsStaffExists"
	ctx, span := st.tracer.Start(
		ctx,
		"StaffRepo.IsStaffExists",
		trace.WithAttributes(
			attribute.String("user.email", logging.RedactEmail(email)),
			attribute.String("user.username", logging.RedactUsername(username)),
			attribute.String("user.barcode", barcode.String()),
		),
	)
	defer span.End()

	query := `
        SELECT
            EXISTS(SELECT 1 FROM users u JOIN staffs s ON u.id = s.user_id WHERE u.email = $1),
            EXISTS(SELECT 1 FROM users u JOIN staffs s ON u.id = s.user_id WHERE u.username = $2),
            EXISTS(SELECT 1 FROM users u JOIN staffs s ON u.id = s.user_id WHERE u.barcode = $3);
    `
	err = st.pool.QueryRow(ctx, query, email, username, barcode).Scan(&emailExists, &usernameExists, &barcodeExists)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to check if staff exists")
		return false, false, false, errorx.Wrap(err, op)
	}

	return emailExists, usernameExists, barcodeExists, nil
}
