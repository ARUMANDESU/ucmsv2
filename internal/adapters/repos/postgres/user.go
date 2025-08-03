package postgres

import (
	"context"
	"os/user"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{
		pool: pool,
	}
}

func (us *UserRepo) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	panic("not implemented") // TODO: Implement
}
