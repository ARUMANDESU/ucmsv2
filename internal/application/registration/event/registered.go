package event

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/logging"
)

var (
	tracer = otel.Tracer("ucms/application/registration/event")
	logger = otelslog.NewLogger("ucms/application/registration/event")
)

type RegistrationRepo interface {
	UpdateRegistration(ctx context.Context, id registration.ID, fn func(context.Context, *registration.Registration) error) error
}

type RegistrationCompletedHandler struct {
	tracer  trace.Tracer
	logger  *slog.Logger
	regRepo RegistrationRepo
}

type RegistrationCompletedHandlerArgs struct {
	Tracer  trace.Tracer
	Logger  *slog.Logger
	RegRepo RegistrationRepo
}

func NewRegistrationCompletedHandler(args RegistrationCompletedHandlerArgs) *RegistrationCompletedHandler {
	if args.Tracer == nil {
		args.Tracer = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}

	return &RegistrationCompletedHandler{
		tracer:  args.Tracer,
		logger:  args.Logger,
		regRepo: args.RegRepo,
	}
}

func (h *RegistrationCompletedHandler) StudentHandle(ctx context.Context, e *user.StudentRegistered) error {
	if e == nil {
		return nil
	}

	l := h.logger.With(
		slog.String("event", "StudentRegistered"),
		slog.String("student.id", e.StudentID.String()),
		slog.String("registration.id", e.RegistrationID.String()),
		slog.String("student.email", logging.RedactEmail(e.Email)),
	)
	ctx, span := h.tracer.Start(ctx, "RegistrationCompletedHandler.StudentHandle",
		trace.WithAttributes(
			attribute.String("student.id", e.StudentID.String()),
			attribute.String("registration.id", e.RegistrationID.String()),
			attribute.String("student.email", logging.RedactEmail(e.Email)),
		))
	defer span.End()

	err := h.regRepo.UpdateRegistration(ctx, e.RegistrationID, func(ctx context.Context, reg *registration.Registration) error {
		err := reg.Complete()
		if err != nil {
			trace.SpanFromContext(ctx).AddEvent("failed to complete registration")
		}
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update registration status to completed")
		l.ErrorContext(ctx, "failed to update registration status to completed", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (h *RegistrationCompletedHandler) StaffHandle(ctx context.Context, e *user.StaffRegistered) error {
	if e == nil {
		return nil
	}

	l := h.logger.With(
		slog.String("event", "StaffRegistered"),
		slog.String("staff.id", e.StaffID.String()),
		slog.String("staff.email", logging.RedactEmail(e.Email)),
		slog.String("registration.id", e.RegistrationID.String()),
	)
	ctx, span := h.tracer.Start(ctx, "RegistrationCompletedHandler.StaffHandle",
		trace.WithAttributes(
			attribute.String("staff.id", e.StaffID.String()),
			attribute.String("staff.email", logging.RedactEmail(e.Email)),
			attribute.String("registration.id", e.RegistrationID.String()),
		))
	defer span.End()

	err := h.regRepo.UpdateRegistration(ctx, e.RegistrationID, func(ctx context.Context, reg *registration.Registration) error {
		err := reg.Complete()
		if err != nil {
			trace.SpanFromContext(ctx).AddEvent("failed to complete registration")
		}
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update registration status to completed")
		l.ErrorContext(ctx, "failed to update registration status to completed", slog.String("error", err.Error()))
		return err
	}

	return nil
}
