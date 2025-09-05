package cmd

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/group"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/i18nx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/logging"
	"gitlab.com/ucmsv2/ucms-backend/pkg/otelx"
)

var (
	ErrBarcodeNotAvailable  = errorx.NewDuplicateEntry().WithKey(i18nx.KeyBarcodeNotAvailable)
	ErrUsernameNotAvailable = errorx.NewDuplicateEntry().WithKey(i18nx.KeyUsernameNotAvailable)
)

type StudentComplete struct {
	Email            string
	VerificationCode string
	Barcode          user.Barcode
	Username         string
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
	const op = "cmd.StudentCompleteHandler.Handle"
	ctx, span := h.tracer.Start(ctx, "StudentCompleteHandler.Handle",
		trace.WithAttributes(
			attribute.String("student.email", logging.RedactEmail(cmd.Email)),
			attribute.String("student.barcode", cmd.Barcode.String()),
			attribute.String("group.id", cmd.GroupID.String()),
		))
	defer span.End()

	emailExists, usernameExists, barcodeExists, err := h.usergetter.IsUserExists(ctx, cmd.Email, cmd.Username, cmd.Barcode)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to check if user exists")
		return errorx.Wrap(err, op)
	}
	if emailExists || usernameExists || barcodeExists {
		errs := make(errorx.I18nErrors, 0)
		if emailExists {
			errs = append(errs, ErrEmailNotAvailable)
		}
		if usernameExists {
			errs = append(errs, ErrUsernameNotAvailable)
		}
		if barcodeExists {
			errs = append(errs, ErrBarcodeNotAvailable)
		}
		otelx.RecordSpanError(span, errs, "validation error: user already exists")
		return errorx.Wrap(errs, op)
	}

	_, err = h.groupgetter.GetGroupByID(ctx, group.ID(cmd.GroupID))
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to get group by ID")
		if errorx.IsNotFound(err) {
			return errorx.NewResourceNotFound(i18nx.FieldGroup).WithCause(err, op)
		}
		return errorx.Wrap(err, op)
	}

	reg, err := h.regRepo.GetRegistrationByEmail(ctx, cmd.Email)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to get registration by email")
		return errorx.Wrap(err, op)
	}

	err = reg.CheckCode(cmd.VerificationCode)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to verify code")
		return errorx.Wrap(err, op)
	}

	student, err := user.RegisterStudent(user.RegisterStudentArgs{
		Barcode:        user.Barcode(cmd.Barcode),
		Username:       cmd.Username,
		RegistrationID: reg.ID(),
		FirstName:      cmd.FirstName,
		LastName:       cmd.LastName,
		AvatarURL:      "",
		Email:          cmd.Email,
		Password:       cmd.Password,
		GroupID:        cmd.GroupID,
	})
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to register student")
		return errorx.Wrap(err, op)
	}

	err = h.studentSaver.SaveStudent(ctx, student)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to save student")
		return errorx.Wrap(err, op)
	}

	return nil
}
