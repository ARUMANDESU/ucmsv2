package ctxs

import (
	"context"

	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
)

type contextKey string

const (
	UserKey = contextKey("userKey")
)

type User struct {
	ID   user.ID
	Role role.Global
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
