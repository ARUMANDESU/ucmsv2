package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/group"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/logging"
)

var (
	ErrMissingVerificationCode    = errorx.NewValidationFieldFailed("verification_code").WithArgs(map[string]any{"Field": "verification_code"})
	ErrMissingBarcode             = errorx.NewValidationFieldFailed("barcode").WithArgs(map[string]any{"Field": "barcode"})
	ErrMissingPassword            = errorx.NewValidationFieldFailed("password").WithArgs(map[string]any{"Field": "password"})
	ErrUserAlreadyExists          = errorx.NewDuplicateEntryWithField("user", "email")
	ErrUserAlreadyExistsByBarcode = errorx.NewDuplicateEntryWithField("user", "barcode")
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

func (c StudentComplete) Validate() error {
	if c.Email == "" {
		return user.ErrMissingEmail
	}
	if c.VerificationCode == "" {
		return ErrMissingVerificationCode
	}
	if c.Barcode == "" {
		return ErrMissingBarcode
	}
	if c.FirstName == "" {
		return user.ErrMissingFirstName
	}
	if c.LastName == "" {
		return user.ErrMissingLastName
	}
	if c.Password == "" {
		return ErrMissingPassword
	}
	if !user.ValidatePasswordManual(c.Password) {
		return user.ErrPasswordNotStrongEnough
	}
	if c.GroupID == uuid.Nil {
		return user.ErrMissingGroupID
	}

	return nil
}

type StudentCompleteHandler struct {
	tracer      trace.Tracer
	logger      *slog.Logger
	usergetter  UserGetter
	groupgetter GroupGetter
	regRepo     Repo
}

type StudentCompleteHandlerArgs struct {
	Trace            trace.Tracer
	Logger           *slog.Logger
	UserGetter       UserGetter
	GroupGetter      GroupGetter
	RegistrationRepo Repo
}

func NewStudentCompleteHandler(args StudentCompleteHandlerArgs) *StudentCompleteHandler {
	if args.Trace == nil {
		args.Trace = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}

	return &StudentCompleteHandler{
		tracer:      args.Trace,
		logger:      args.Logger,
		usergetter:  args.UserGetter,
		groupgetter: args.GroupGetter,
		regRepo:     args.RegistrationRepo,
	}
}

func (h *StudentCompleteHandler) Handle(ctx context.Context, cmd StudentComplete) error {
	ctx, span := h.tracer.Start(ctx, "StudentCompleteHandler.Handle")
	defer span.End()

	if err := cmd.Validate(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid command parameters")
		return fmt.Errorf("invalid command parameters: %w", err)
	}

	u, err := h.usergetter.GetUserByEmail(ctx, cmd.Email)
	if err != nil && !errorx.IsNotFound(err) {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get user by email")
		return fmt.Errorf("failed to get user by email: %w", err)
	}
	if u != nil {
		span.RecordError(ErrUserAlreadyExists)
		span.SetStatus(codes.Error, "User already exists by email")
		return ErrUserAlreadyExists
	}

	u, err = h.usergetter.GetUserByID(ctx, user.ID(cmd.Barcode))
	if err != nil && !errorx.IsNotFound(err) {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get user by barcode")
		return fmt.Errorf("failed to get user by barcode: %w", err)
	}
	if u != nil {
		span.RecordError(ErrUserAlreadyExistsByBarcode)
		span.SetStatus(codes.Error, "User already exists by barcode")
		return ErrUserAlreadyExistsByBarcode
	}

	_, err = h.groupgetter.GetGroupByID(ctx, group.ID(cmd.GroupID))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get group by ID")
		return fmt.Errorf("failed to get group by ID: %w", err)
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
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update registration")
		return fmt.Errorf("failed to update registration: %w", err)
	}

	return nil
}
