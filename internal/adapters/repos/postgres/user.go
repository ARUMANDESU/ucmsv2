package postgres

import (
	"context"
	"errors"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/otelx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/postgres"
	"gitlab.com/ucmsv2/ucms-backend/pkg/watermillx"
)

const insertUserQuery = ` INSERT INTO users (id, barcode, username, role_id, email, first_name, last_name, avatar_source, avatar_external, avatar_s3_key, pass_hash, created_at, updated_at)
    VALUES ($1, $2, $3, (SELECT id FROM global_roles WHERE name = $4), $5, $6, $7, $8, $9, $10, $11, $12, $13);`

type UserRepo struct {
	tracer  trace.Tracer
	logger  *slog.Logger
	pool    *pgxpool.Pool
	wlogger watermill.LoggerAdapter
}

// NewUserRepo creates a new instance of UserRepo.
//
// WARNING: panics if pool is nil
func NewUserRepo(pool *pgxpool.Pool, t trace.Tracer, l *slog.Logger) *UserRepo {
	if pool == nil {
		panic("pgxpool.Pool cannot be nil")
	}
	if t == nil {
		t = tracer
	}
	if l == nil {
		l = logger
	}

	return &UserRepo{
		tracer: t,
		logger: l,
		pool:   pool,
	}
}

func (r *UserRepo) SaveUser(ctx context.Context, u *user.User) error {
	const op = "postgres.UserRepo.SaveUser"
	ctx, span := r.tracer.Start(ctx, "UserRepo.SaveUser")
	defer span.End()

	err := postgres.WithTx(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		dto := DomainToUserDTO(u)
		res, err := tx.Exec(ctx, insertUserQuery,
			dto.ID,
			dto.Barcode,
			dto.Username,
			u.Role().String(),
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
			return errorx.Wrap(err, op)
		}
		if res.RowsAffected() == 0 {
			otelx.RecordSpanError(span, err, "no rows affected while inserting user")
			return errorx.Wrap(ErrNoRowsAffected, op)
		}

		events := u.GetUncommittedEvents()
		if len(events) > 0 {
			if err := watermillx.Publish(ctx, tx, r.wlogger, events...); err != nil {
				otelx.RecordSpanError(span, err, "failed to publish events")
				return errorx.Wrap(err, op)
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

func (r *UserRepo) UpdateUser(
	ctx context.Context,
	id user.ID,
	fn func(ctx context.Context, u *user.User) error,
) error {
	const op = "postgres.UserRepo.UpdateUser"
	ctx, span := r.tracer.Start(ctx, "UserRepo.UpdateUser")
	defer span.End()
	if fn == nil {
		otelx.RecordSpanError(span, ErrNilFunc, "update function cannot be nil")
		return ErrNilFunc
	}
	err := postgres.WithTx(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		query := `
        SELECT  u.id, u.barcode, u.username, u.role_id,
                u.first_name, u.last_name, 
				u.avatar_source, u.avatar_external, u.avatar_s3_key,
                u.email, u.pass_hash, u.created_at, u.updated_at,
                gr.id, gr.name
        FROM users u JOIN global_roles gr ON u.role_id = gr.id
        WHERE u.id = $1;
    `

		var dto UserDTO
		var roleDTO GlobalRoleDTO
		err := r.pool.QueryRow(ctx, query, id).
			Scan(
				&dto.ID, &dto.Barcode, &dto.Username, &dto.RoleID,
				&dto.FirstName, &dto.LastName,
				&dto.AvatarSource, &dto.AvatarExternal, &dto.AvatarS3Key,
				&dto.Email, &dto.Passhash, &dto.CreatedAt, &dto.UpdatedAt,
				&roleDTO.ID, &roleDTO.Name,
			)
		if err != nil {
			otelx.RecordSpanError(span, err, "failed to get user by id")
			if errors.Is(err, pgx.ErrNoRows) {
				return errorx.NewNotFound().WithCause(err, op)
			}
			return errorx.Wrap(err, op)
		}

		u := UserToDomain(dto, roleDTO)

		fnerr := fn(ctx, u)
		if fnerr != nil && !errorx.IsPersistable(fnerr) {
			otelx.RecordSpanError(span, fnerr, "update function returned an error and cannot continue")
			return errorx.Wrap(fnerr, op)
		}

		dto = DomainToUserDTO(u)

		updateQuery := `
		UPDATE users
		SET barcode = $2, username = $3, role_id = (SELECT id FROM global_roles WHERE name = $4),
			first_name = $5, last_name = $6,
			avatar_source = $7, avatar_external = $8, avatar_s3_key = $9,
			email = $10, pass_hash = $11, updated_at = $12
		WHERE id = $1;
		`

		res, err := tx.Exec(ctx, updateQuery,
			dto.ID,
			dto.Barcode,
			dto.Username,
			u.Role().String(),
			dto.FirstName,
			dto.LastName,
			dto.AvatarSource,
			dto.AvatarExternal,
			dto.AvatarS3Key,
			dto.Email,
			dto.Passhash,
			dto.UpdatedAt,
		)
		if err != nil {
			otelx.RecordSpanError(span, err, "failed to update user")
			return errorx.Wrap(err, op)
		}
		if res.RowsAffected() == 0 {
			otelx.RecordSpanError(span, err, "no rows affected while updating user")
			return errorx.Wrap(ErrNoRowsAffected, op)
		}

		events := u.GetUncommittedEvents()
		if len(events) > 0 {
			if err := watermillx.Publish(ctx, tx, r.wlogger, events...); err != nil {
				otelx.RecordSpanError(span, err, "failed to publish events")
				return errorx.Wrap(err, op)
			}
		}

		if fnerr != nil && errorx.IsPersistable(fnerr) {
			otelx.RecordSpanError(span, fnerr, "update function returned an error but is allowed to continue")
			return errorx.Wrap(fnerr, op)
		}

		return nil
	})
	if err != nil {
		otelx.RecordSpanError(span, err, "transaction to update user failed")
		return err
	}

	return nil
}

func (r *UserRepo) GetUserByID(ctx context.Context, id user.ID) (*user.User, error) {
	const op = "postgres.UserRepo.GetUserByID"
	ctx, span := r.tracer.Start(ctx, "UserRepo.GetUserByID")
	defer span.End()

	query := `
        SELECT  u.id, u.barcode, u.username, u.role_id,
                u.first_name, u.last_name, 
				u.avatar_source, u.avatar_external, u.avatar_s3_key,
                u.email, u.pass_hash, u.created_at, u.updated_at,
                gr.id, gr.name
        FROM users u JOIN global_roles gr ON u.role_id = gr.id
        WHERE u.id = $1;
    `

	var dto UserDTO
	var roleDTO GlobalRoleDTO
	err := r.pool.QueryRow(ctx, query, id).
		Scan(
			&dto.ID, &dto.Barcode, &dto.Username, &dto.RoleID,
			&dto.FirstName, &dto.LastName,
			&dto.AvatarSource, &dto.AvatarExternal, &dto.AvatarS3Key,
			&dto.Email, &dto.Passhash, &dto.CreatedAt, &dto.UpdatedAt,
			&roleDTO.ID, &roleDTO.Name,
		)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to get user by id")
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorx.NewNotFound().WithCause(err, op)
		}
		return nil, errorx.Wrap(err, op)
	}

	return UserToDomain(dto, roleDTO), nil
}

func (r *UserRepo) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	ctx, span := r.tracer.Start(ctx, "UserRepo.GetUserByEmail")
	defer span.End()

	query := `
        SELECT  u.id, u.barcode, u.username, u.role_id, 
                u.first_name, u.last_name, 
				u.avatar_source, u.avatar_external, u.avatar_s3_key,
                u.email, u.pass_hash, u.created_at, u.updated_at,
                gr.id, gr.name
        FROM users u JOIN global_roles gr ON u.role_id = gr.id
        WHERE email = $1;
    `

	var dto UserDTO
	var roleDTO GlobalRoleDTO
	err := r.pool.QueryRow(ctx, query, email).
		Scan(
			&dto.ID, &dto.Barcode, &dto.Username, &dto.RoleID,
			&dto.FirstName, &dto.LastName,
			&dto.AvatarSource, &dto.AvatarExternal, &dto.AvatarS3Key,
			&dto.Email, &dto.Passhash, &dto.CreatedAt, &dto.UpdatedAt,
			&roleDTO.ID, &roleDTO.Name,
		)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to get user by email")
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorx.NewNotFound().WithCause(err, "get_user_by_email")
		}
		return nil, err
	}

	return UserToDomain(dto, roleDTO), nil
}

func (r *UserRepo) GetUserByBarcode(ctx context.Context, barcode user.Barcode) (*user.User, error) {
	const op = "postgres.UserRepo.GetUserByBarcode"
	ctx, span := r.tracer.Start(ctx, "UserRepo.GetUserByBarcode")
	defer span.End()

	query := `
        SELECT  u.id, u.barcode, u.username, u.role_id,
                u.first_name, u.last_name, 
				u.avatar_source, u.avatar_external, u.avatar_s3_key,
                u.email, u.pass_hash, u.created_at, u.updated_at,
                gr.id, gr.name
        FROM users u JOIN global_roles gr ON u.role_id = gr.id
        WHERE u.barcode = $1;
    `

	var dto UserDTO
	var roleDTO GlobalRoleDTO
	err := r.pool.QueryRow(ctx, query, barcode).
		Scan(
			&dto.ID, &dto.Barcode, &dto.Username, &dto.RoleID,
			&dto.FirstName, &dto.LastName,
			&dto.AvatarSource, &dto.AvatarExternal, &dto.AvatarS3Key,
			&dto.Email, &dto.Passhash, &dto.CreatedAt, &dto.UpdatedAt,
			&roleDTO.ID, &roleDTO.Name,
		)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to get user by barcode")
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorx.NewNotFound().WithCause(err, op)
		}
		return nil, errorx.Wrap(err, op)
	}

	return UserToDomain(dto, roleDTO), nil
}

func (r *UserRepo) IsUserExists(
	ctx context.Context,
	email, username string,
	barcode user.Barcode,
) (emailExists, usernameExists, barcodeExists bool, err error) {
	const op = "postgres.UserRepo.IsUserExists"
	ctx, span := r.tracer.Start(ctx, "UserRepo.IsUserExists")
	defer span.End()

	query := `
        SELECT  EXISTS(SELECT 1 FROM users WHERE email = $1),
                EXISTS(SELECT 1 FROM users WHERE username = $2),
                EXISTS(SELECT 1 FROM users WHERE barcode = $3);
    `

	err = r.pool.QueryRow(ctx, query, email, username, barcode).
		Scan(&emailExists, &usernameExists, &barcodeExists)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to check if user exists")
		return false, false, false, errorx.Wrap(err, op)
	}

	return emailExists, usernameExists, barcodeExists, nil
}
