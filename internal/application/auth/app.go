package auth

import (
	"context"

	"github.com/ARUMANDESU/ucms/internal/domain/email"
)

type Repo interface {
	GetVerificationCode(ctx context.Context, email string) (email.EmailVerificationCode, error)
	SaveVerificationCode(ctx context.Context, code email.EmailVerificationCode) error
}

type Application struct {
}

func (a *Application) RequestEmailVerificationCode(ctx context.Context, email string) error {

	return nil
}
