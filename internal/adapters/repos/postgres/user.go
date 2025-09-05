package postgres

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/otelx"
)

const insertUserQuery = `
    INSERT INTO users (id, barcode, username, role_id, email, first_name, last_name, avatar_url, pass_hash, created_at, updated_at)
    VALUES ($1, $2, $3, (SELECT id FROM global_roles WHERE name = $4), $5, $6, $7, $8, $9, $10, $11);`

type UserRepo struct {
	tracer trace.Tracer
	logger *slog.Logger
	pool   *pgxpool.Pool
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

func (r *UserRepo) GetUserByID(ctx context.Context, id user.ID) (*user.User, error) {
	const op = "postgres.UserRepo.GetUserByID"
	ctx, span := r.tracer.Start(ctx, "UserRepo.GetUserByID")
	defer span.End()

	query := `
        SELECT  u.id, u.barcode, u.username, u.role_id,
                u.first_name, u.last_name, u.avatar_url,
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
			&dto.FirstName, &dto.LastName, &dto.AvatarURL,
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
                u.first_name, u.last_name, u.avatar_url, 
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
			&dto.FirstName, &dto.LastName, &dto.AvatarURL,
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
                u.first_name, u.last_name, u.avatar_url, 
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
			&dto.FirstName, &dto.LastName, &dto.AvatarURL,
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
