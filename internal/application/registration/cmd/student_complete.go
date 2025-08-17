package cmd

import (
	"context"
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

var ErrBarcodeNotAvailable = errorx.NewDuplicateEntry().WithKey("error_barcode_not_available")

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
	ctx, span := h.tracer.Start(ctx, "StudentCompleteHandler.Handle",
		trace.WithAttributes(
			attribute.String("student.email", logging.RedactEmail(cmd.Email)),
			attribute.String("student.barcode", cmd.Barcode),
			attribute.String("group.id", cmd.GroupID.String()),
		))
	defer span.End()

	u, err := h.usergetter.GetUserByEmail(ctx, cmd.Email)
	if err != nil && !errorx.IsNotFound(err) {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get user by email")
		return err
	}
	if u != nil {
		span.RecordError(ErrEmailNotAvailable)
		span.SetStatus(codes.Error, "user already exists by email")
		return ErrEmailNotAvailable
	}

	u, err = h.usergetter.GetUserByID(ctx, user.ID(cmd.Barcode))
	if err != nil && !errorx.IsNotFound(err) {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get user by barcode")
		return err
	}
	if u != nil {
		span.RecordError(ErrBarcodeNotAvailable)
		span.SetStatus(codes.Error, "user already exists by barcode")
		return ErrBarcodeNotAvailable
	}

	_, err = h.groupgetter.GetGroupByID(ctx, group.ID(cmd.GroupID))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get group by id")
		return err
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
			span.AddEvent("failed to complete student registration")
			return err
		}
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update registration")
		return err
	}

	return nil
}
