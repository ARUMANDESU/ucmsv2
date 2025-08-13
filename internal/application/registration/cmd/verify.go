package cmd

import (
	"context"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
)

var ErrOKAlreadyVerified = errorx.NewAlreadyProcessed().WithHTTPCode(http.StatusOK)

type Verify struct {
	Email string
	Code  string
}

type VerifyHandler struct {
	tracer trace.Tracer
	logger *slog.Logger
	repo   Repo
}

type VerifyHandlerArgs struct {
	Tracer           trace.Tracer
	Logger           *slog.Logger
	RegistrationRepo Repo
}

func NewVerifyHandler(args VerifyHandlerArgs) *VerifyHandler {
	if args.Tracer == nil {
		args.Tracer = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}

	return &VerifyHandler{
		tracer: args.Tracer,
		logger: args.Logger,
		repo:   args.RegistrationRepo,
	}
}

func (h *VerifyHandler) Handle(ctx context.Context, cmd Verify) error {
	ctx, span := h.tracer.Start(ctx, "VerifyHandler.Handle")
	defer span.End()

	h.logger.Debug("VerifyHandler.Handle called", "email", cmd.Email, "code", cmd.Code)

	if cmd.Email == "" || cmd.Code == "" {
		span.RecordError(errorx.ErrInvalidInput)
		span.SetStatus(codes.Error, "email or code is empty")
		return errorx.ErrInvalidInput
	}

	return h.repo.UpdateRegistrationByEmail(ctx, cmd.Email, func(ctx context.Context, r *registration.Registration) error {
		span := trace.SpanFromContext(ctx)

		if r.IsStatus(registration.StatusVerified) {
			span.SetStatus(codes.Ok, "registration already verified")
			return ErrOKAlreadyVerified
		}

		if err := r.VerifyCode(cmd.Code); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to verify code")
			return err
		}

		return nil
	})
}
