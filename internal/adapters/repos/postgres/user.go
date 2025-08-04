package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ARUMANDESU/ucms/internal/adapters/repos"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{
		pool: pool,
	}
}

func (r *UserRepo) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	query := `
        SELECT u.id, u.role_id, u.first_name, u.last_name, u.avatar_url, u.email, u.pass_hash, u.created_at, u.updated_at,
               gr.id, gr.name
        FROM users u JOIN global_roles gr ON u.role_id = gr.id
        WHERE email = $1;
    `

	var dto UserDTO
	var roleDTO GlobalRoleDTO
	err := r.pool.QueryRow(ctx, query, email).
		Scan(&dto.ID, &dto.RoleID, &dto.FirstName, &dto.LastName, &dto.AvatarURL, &dto.Email, &dto.Passhash, &dto.CreatedAt, &dto.UpdatedAt,
			&roleDTO.ID, &roleDTO.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repos.ErrNotFound
		}
		return nil, err
	}

	return UserToDomain(dto, roleDTO), nil
}
