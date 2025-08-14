package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/logging"
)

type ResendCode struct {
	Email string
}

type ResendCodeHandler struct {
	tracer     trace.Tracer
	logger     *slog.Logger
	repo       Repo
	usergetter UserGetter
}

type ResendCodeHandlerArgs struct {
	Tracer     trace.Tracer
	Logger     *slog.Logger
	Repo       Repo
	UserGetter UserGetter
}

func NewResendCodeHandler(args ResendCodeHandlerArgs) *ResendCodeHandler {
	if args.Tracer == nil {
		args.Tracer = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}

	return &ResendCodeHandler{
		tracer:     args.Tracer,
		logger:     args.Logger,
		repo:       args.Repo,
		usergetter: args.UserGetter,
	}
}

func (h *ResendCodeHandler) Handle(ctx context.Context, cmd *ResendCode) error {
	ctx, span := h.tracer.Start(ctx, "ResendCodeHandler.Handle")
	defer span.End()

	h.logger.DebugContext(ctx, "ResendCodeHandler.Handle called", slog.String("email", cmd.Email))

	if cmd.Email == "" {
		err := user.ErrMissingEmail
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetAttributes(attribute.String("email", logging.RedactEmail(cmd.Email)))

	user, err := h.usergetter.GetUserByEmail(ctx, cmd.Email)
	if err != nil && !errorx.IsNotFound(err) {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get user by email")
		return fmt.Errorf("failed to get user by email: %w", err)
	}
	if user != nil {
		span.RecordError(errorx.NewDuplicateEntryWithField("user", "email"))
		span.SetStatus(codes.Error, "User already exists")
		return errorx.NewDuplicateEntryWithField("user", "email")
	}
	span.AddEvent("User not found, proceeding with registration")

	err = h.repo.UpdateRegistrationByEmail(ctx, cmd.Email, func(ctx context.Context, r *registration.Registration) error {
		span := trace.SpanFromContext(ctx)
		err := r.ResendCode()
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to resend code")
			return fmt.Errorf("failed to resend code: %w", err)
		}
		span.AddEvent("Code resent successfully")
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update registration")
		return fmt.Errorf("failed to update registration: %w", err)
	}

	return nil
}
