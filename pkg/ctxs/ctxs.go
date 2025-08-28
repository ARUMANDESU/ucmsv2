package ctxs

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
)

const (
	TxKey   = "pgxTxKey"
	UserKey = "userKey"
)

type User struct {
	ID   user.ID
	Role role.Global
}

func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, TxKey, tx)
}

func Tx(ctx context.Context) (pgx.Tx, bool) {
	val := ctx.Value(TxKey)
	if val == nil {
		return nil, false
	}

	tx, ok := val.(pgx.Tx)
	return tx, ok
}

func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, UserKey, user)
}

func UserFromCtx(ctx context.Context) (*User, bool) {
	val := ctx.Value(UserKey)
	if val == nil {
		return nil, false
	}

	user, ok := val.(*User)
	return user, ok
}
