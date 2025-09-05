package ctxs

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/valueobject/roles"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
)

var (
	ErrNotFoundInContext    = errors.New("not found in context")
	ErrInvalidTypeInContext = errors.New("invalid type in context")
)

type contextKey string

const (
	UserKey = contextKey("userKey")
)

type User struct {
	ID   user.ID
	Role roles.Global
}

func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, UserKey, user)
}

func UserFromCtx(ctx context.Context) (*User, error) {
	const op = "ctxs.UserFromCtx"
	val := ctx.Value(UserKey)
	if val == nil {
		return nil, errorx.NewInternalError().WithCause(ErrNotFoundInContext, op)
	}

	user, ok := val.(*User)
	if !ok {
		return nil, errorx.NewInternalError().WithCause(ErrInvalidTypeInContext, op)
	}
	return user, nil
}

func (u User) SetSpanAttrs(span trace.Span) {
	if span == nil {
		return
	}
	span.SetAttributes(
		attribute.String("user.id", u.ID.String()),
		attribute.String("user.role", u.Role.String()),
	)
}
