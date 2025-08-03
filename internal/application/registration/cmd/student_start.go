package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/ARUMANDESU/ucms/internal/adapters/repos"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/pkg/apperr"
	"github.com/ARUMANDESU/ucms/pkg/env"
)

type StartStudent struct {
	Email string
}

type StartStudentHandler struct {
	mode       env.Mode
	repo       Repo
	usergetter UserGetter
}

type StartStudentHandlerArgs struct {
	Mode       env.Mode
	Repo       Repo
	UserGetter UserGetter
}

func NewStartStudentHandler(args StartStudentHandlerArgs) *StartStudentHandler {
	return &StartStudentHandler{
		mode:       args.Mode,
		repo:       args.Repo,
		usergetter: args.UserGetter,
	}
}

func (h *StartStudentHandler) Handle(ctx context.Context, cmd StartStudent) error {
	user, err := h.usergetter.GetUserByEmail(ctx, cmd.Email)
	if err != nil && !errors.Is(err, repos.ErrNotFound) {
		return fmt.Errorf("failed to get user by email: %w", err)
	}
	if user != nil {
		return apperr.NewConflict("user with this email already exists")
	}

	reg, err := h.repo.GetRegistrationByEmail(ctx, cmd.Email)
	if err != nil && !errors.Is(err, repos.ErrNotFound) {
		return fmt.Errorf("failed to get registration by email: %w", err)
	}

	if reg == nil || errors.Is(err, repos.ErrNotFound) {
		reg, err = registration.NewRegistration(cmd.Email, h.mode)
		if err != nil {
			return fmt.Errorf("failed to create new registration: %w", err)
		}

		err = h.repo.SaveRegistration(ctx, reg)
		if err != nil {
			return fmt.Errorf("failed to save registration: %w", err)
		}

		return nil
	}

	if reg.IsCompleted() {
		return apperr.NewConflict("user with this email is already registered")
	}

	err = h.repo.UpdateRegistration(ctx, reg.ID(), func(ctx context.Context, r *registration.Registration) error {
		return r.ResendCode()
	})
	if err != nil {
		return fmt.Errorf("failed to update registration: %w", err)
	}

	return nil
}
