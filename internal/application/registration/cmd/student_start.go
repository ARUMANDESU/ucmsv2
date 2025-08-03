package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/adapters/repos"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/pkg/apperr"
	"github.com/ARUMANDESU/ucms/pkg/env"
	"github.com/ARUMANDESU/ucms/pkg/logging"
)

var (
	tracer = otel.Tracer("ucms/application/registration/cmd")
	logger = otelslog.NewLogger("ucms/application/registration/cmd")
)

type StartStudent struct {
	Email string
}

type StartStudentHandler struct {
	tracer     trace.Tracer
	logger     *slog.Logger
	mode       env.Mode
	repo       Repo
	usergetter UserGetter
}

type StartStudentHandlerArgs struct {
	Tracer     trace.Tracer
	Logger     *slog.Logger
	Mode       env.Mode
	Repo       Repo
	UserGetter UserGetter
}

func NewStartStudentHandler(args StartStudentHandlerArgs) *StartStudentHandler {
	if args.Tracer == nil {
		args.Tracer = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}

	return &StartStudentHandler{
		tracer:     args.Tracer,
		logger:     args.Logger,
		mode:       args.Mode,
		repo:       args.Repo,
		usergetter: args.UserGetter,
	}
}

func (h *StartStudentHandler) Handle(ctx context.Context, cmd StartStudent) error {
	ctx, span := h.tracer.Start(ctx, "StartStudentHandler.Handle")
	defer span.End()

	redactedEmail := logging.RedactEmail(cmd.Email)
	span.SetAttributes(attribute.String("student.email", redactedEmail))

	h.logger.DebugContext(ctx, "starting student registration")

	user, err := h.usergetter.GetUserByEmail(ctx, cmd.Email)
	if err != nil && !errors.Is(err, repos.ErrNotFound) {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get user by email")
		return fmt.Errorf("failed to get user by email: %w", err)
	}
	if user != nil {
		span.RecordError(apperr.NewConflict("user with this email already exists"))
		span.SetStatus(codes.Error, "User already exists")
		return apperr.NewConflict("user with this email already exists")
	}
	span.AddEvent("User not found, proceeding with registration")

	reg, err := h.repo.GetRegistrationByEmail(ctx, cmd.Email)
	if err != nil && !errors.Is(err, repos.ErrNotFound) {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get registration by email")
		return fmt.Errorf("failed to get registration by email: %w", err)
	}
	if errors.Is(err, repos.ErrNotFound) {
		reg, err = registration.NewRegistration(cmd.Email, h.mode)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to create new registration")
			return fmt.Errorf("failed to create new registration: %w", err)
		}

		err = h.repo.SaveRegistration(ctx, reg)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to save registration")
			return fmt.Errorf("failed to save registration: %w", err)
		}
		span.AddEvent("Registration saved successfully",
			trace.WithAttributes(
				attribute.String("registration.id", reg.ID().String()),
				attribute.String("registration.status", reg.Status().String()),
			),
		)

		return nil
	}

	if reg.IsCompleted() {
		return apperr.NewConflict("user with this email is already registered")
	}

	span.AddEvent("Registration found: proceeding with verification code resend")

	err = h.repo.UpdateRegistration(ctx, reg.ID(), func(ctx context.Context, r *registration.Registration) error {
		err := r.ResendCode()
		if err != nil {
			trace.SpanFromContext(ctx).AddEvent("resend verification code failed")
			return fmt.Errorf("failed to resend verification code: %w", err)
		}
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update registration")
		return fmt.Errorf("failed to update registration: %w", err)
	}

	return nil
}
