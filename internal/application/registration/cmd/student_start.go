package cmd

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/pkg/env"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/i18nx"
	"github.com/ARUMANDESU/ucms/pkg/logging"
	"github.com/ARUMANDESU/ucms/pkg/otelx"
)

var ErrEmailNotAvailable = errorx.NewDuplicateEntry().WithKey(i18nx.KeyEmailNotAvailable)

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
	ctx, span := h.tracer.Start(
		ctx,
		"StartStudentHandler.Handle",
		trace.WithAttributes(attribute.String("student.email", logging.RedactEmail(cmd.Email))),
	)
	defer span.End()

	user, err := h.usergetter.GetUserByEmail(ctx, cmd.Email)
	if err != nil && !errorx.IsNotFound(err) {
		otelx.RecordSpanError(span, err, "failed to get user by email")
		return err
	}
	if user != nil {
		otelx.RecordSpanError(span, ErrEmailNotAvailable, "user already exists with this email")
		return ErrEmailNotAvailable
	}
	span.AddEvent("user not found, proceeding with registration")

	reg, err := h.repo.GetRegistrationByEmail(ctx, cmd.Email)
	if err != nil && !errorx.IsNotFound(err) {
		otelx.RecordSpanError(span, err, "failed to get registration by email")
		return err
	}
	if errorx.IsNotFound(err) {
		reg, err = registration.NewRegistration(cmd.Email, h.mode)
		if err != nil {
			otelx.RecordSpanError(span, err, "failed to create new registration")
			return err
		}

		err = h.repo.SaveRegistration(ctx, reg)
		if err != nil {
			otelx.RecordSpanError(span, err, "failed to save new registration")
			return err
		}
		span.AddEvent("registration saved successfully",
			trace.WithAttributes(
				attribute.String("registration.id", reg.ID().String()),
				attribute.String("registration.status", reg.Status().String()),
			),
		)

		return nil
	}

	if reg.IsCompleted() {
		otelx.RecordSpanError(span, ErrEmailNotAvailable, "registration already completed with this email")
		return ErrEmailNotAvailable
	}

	err = h.repo.UpdateRegistration(ctx, reg.ID(), func(ctx context.Context, r *registration.Registration) error {
		err := r.ResendCode()
		if err != nil {
			trace.SpanFromContext(ctx).AddEvent("resend verification code failed")
			return err
		}

		return nil
	})
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to resend code for existing registration")
		return err
	}

	return nil
}
