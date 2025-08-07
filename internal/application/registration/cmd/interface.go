package cmd

import (
	"context"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
)

type Repo interface {
	GetRegistrationByEmail(ctx context.Context, email string) (*registration.Registration, error)
	SaveRegistration(ctx context.Context, r *registration.Registration) error
	UpdateRegistration(ctx context.Context, id registration.ID, fn func(context.Context, *registration.Registration) error) error
	UpdateRegistrationByEmail(ctx context.Context, email string, fn func(context.Context, *registration.Registration) error) error
}

type UserGetter interface {
	GetUserByEmail(ctx context.Context, email string) (*user.User, error)
}

// type StaffSignUpTokenGetter interface {
//     GetStaffSignUpToken(ctx context.Context, token string) (*)
// }
