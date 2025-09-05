package cmd

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/registration"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/logging"
	"gitlab.com/ucmsv2/ucms-backend/pkg/otelx"
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

func (h *ResendCodeHandler) Handle(ctx context.Context, cmd ResendCode) error {
	const op = "cmd.ResendCodeHandler.Handle"
	ctx, span := h.tracer.Start(ctx, "ResendCodeHandler.Handle",
		trace.WithAttributes(
			attribute.String("email", logging.RedactEmail(cmd.Email)),
		))
	defer span.End()

	user, err := h.usergetter.GetUserByEmail(ctx, cmd.Email)
	if err != nil && !errorx.IsNotFound(err) {
		otelx.RecordSpanError(span, err, "failed to get user by email")
		return errorx.Wrap(err, op)
	}
	if user != nil {
		otelx.RecordSpanError(span, ErrEmailNotAvailable, "user already exists with this email")
		return errorx.Wrap(ErrEmailNotAvailable, op)
	}
	span.AddEvent("user not found, proceeding to resend code")

	err = h.repo.UpdateRegistrationByEmail(ctx, cmd.Email, func(ctx context.Context, r *registration.Registration) error {
		span := trace.SpanFromContext(ctx)
		otelx.SetSpanAttrs(span, map[string]any{
			"registration.id":     r.ID().String(),
			"registration.status": r.Status().String(),
		})
		err := r.ResendCode()
		if err != nil {
			span.AddEvent("failed to resend code")
			return err
		}
		span.AddEvent("code resent successfully")
		return nil
	})
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to update registration by email")
		return errorx.Wrap(err, op)
	}

	return nil
}
