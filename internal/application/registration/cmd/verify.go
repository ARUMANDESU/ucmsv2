package cmd

import (
	"context"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/registration"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/logging"
	"gitlab.com/ucmsv2/ucms-backend/pkg/otelx"
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
	const op = "cmd.VerifyHandler.Handle"
	ctx, span := h.tracer.Start(ctx, "VerifyHandler.Handle",
		trace.WithAttributes(attribute.String("email", logging.RedactEmail(cmd.Email))),
	)
	defer span.End()

	err := h.repo.UpdateRegistrationByEmail(ctx, cmd.Email, func(ctx context.Context, r *registration.Registration) error {
		span := trace.SpanFromContext(ctx)

		if r.IsStatus(registration.StatusVerified) {
			span.AddEvent("registration already verified")
			return errorx.Wrap(ErrOKAlreadyVerified, op)
		}

		if err := r.VerifyCode(cmd.Code); err != nil {
			span.AddEvent("failed to verify registration code")
			return errorx.Wrap(err, op)
		}

		return nil
	})
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to update registration by email")
		return errorx.Wrap(err, op)
	}

	return nil
}
