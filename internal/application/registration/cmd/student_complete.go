package cmd

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/group"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/logging"
)

var ErrBarcodeNotAvailable = errorx.NewDuplicateEntry().WithKey("error_barcode_not_available")

type StudentComplete struct {
	Email            string
	VerificationCode string
	Barcode          user.Barcode
	FirstName        string
	LastName         string
	Password         string
	GroupID          group.ID
}

type StudentCompleteHandler struct {
	tracer       trace.Tracer
	logger       *slog.Logger
	usergetter   UserGetter
	groupgetter  GroupGetter
	regRepo      Repo
	studentSaver StudentSaver
}

type StudentCompleteHandlerArgs struct {
	Trace            trace.Tracer
	Logger           *slog.Logger
	UserGetter       UserGetter
	GroupGetter      GroupGetter
	RegistrationRepo Repo
	StudentSaver     StudentSaver
}

func NewStudentCompleteHandler(args StudentCompleteHandlerArgs) *StudentCompleteHandler {
	if args.Trace == nil {
		args.Trace = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}

	return &StudentCompleteHandler{
		tracer:       args.Trace,
		logger:       args.Logger,
		usergetter:   args.UserGetter,
		groupgetter:  args.GroupGetter,
		regRepo:      args.RegistrationRepo,
		studentSaver: args.StudentSaver,
	}
}

func (h *StudentCompleteHandler) Handle(ctx context.Context, cmd StudentComplete) error {
	ctx, span := h.tracer.Start(ctx, "StudentCompleteHandler.Handle",
		trace.WithAttributes(
			attribute.String("student.email", logging.RedactEmail(cmd.Email)),
			attribute.String("student.barcode", cmd.Barcode.String()),
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

	u, err = h.usergetter.GetUserByBarcode(ctx, user.Barcode(cmd.Barcode))
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
		if errorx.IsNotFound(err) {
			return errorx.NewResourceNotFound("group").WithCause(err)
		}
		return err
	}

	reg, err := h.regRepo.GetRegistrationByEmail(ctx, cmd.Email)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get registration by email")
		return err
	}

	err = reg.CheckCode(cmd.VerificationCode)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to check verification code")
		return err
	}

	student, err := user.RegisterStudent(user.RegisterStudentArgs{
		Barcode:        user.Barcode(cmd.Barcode),
		RegistrationID: reg.ID(),
		FirstName:      cmd.FirstName,
		LastName:       cmd.LastName,
		AvatarURL:      "",
		Email:          cmd.Email,
		Password:       cmd.Password,
		GroupID:        cmd.GroupID,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to register student")
		return err
	}

	err = h.studentSaver.SaveStudent(ctx, student)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to save student")
		return err
	}

	return nil
}
