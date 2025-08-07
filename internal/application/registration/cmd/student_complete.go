package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/adapters/repos"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/pkg/apperr"
	"github.com/ARUMANDESU/ucms/pkg/logging"
)

type StudentComplete struct {
	Email            string
	VerificationCode string
	Barcode          string
	FirstName        string
	LastName         string
	Password         string
	GroupID          uuid.UUID
}

type StudentCompleteHandler struct {
	tracer     trace.Tracer
	logger     *slog.Logger
	usergetter UserGetter
	regRepo    Repo
}

type StudentCompleteHandlerArgs struct {
	Trace            trace.Tracer
	Logger           *slog.Logger
	UserGetter       UserGetter
	RegistrationRepo Repo
}

func NewStudentCompleteHandler(args StudentCompleteHandlerArgs) *StudentCompleteHandler {
	return &StudentCompleteHandler{
		tracer:     args.Trace,
		logger:     args.Logger,
		usergetter: args.UserGetter,
		regRepo:    args.RegistrationRepo,
	}
}

func (h *StudentCompleteHandler) Handle(ctx context.Context, cmd StudentComplete) error {
	ctx, span := h.tracer.Start(ctx, "StudentCompleteHandler.Handle")
	defer span.End()

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

	err = h.regRepo.UpdateRegistrationByEmail(ctx, cmd.Email, func(ctx context.Context, r *registration.Registration) error {
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(
			attribute.String("registration.id", r.ID().String()),
			attribute.String("registration.email", logging.RedactEmail(r.Email())),
		)

		err := r.CompleteStudentRegistration(registration.StudentArgs{
			VerificationCode: cmd.VerificationCode,
			Barcode:          cmd.Barcode,
			FirstName:        cmd.FirstName,
			LastName:         cmd.LastName,
			Password:         cmd.Password,
			GroupID:          cmd.GroupID,
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to complete student registration")
			return fmt.Errorf("failed to complete student registration: %w", err)
		}
		return nil
	})

	return nil
}
