package cmd

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/staffinvitation"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/i18nx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/otelx"
)

var (
	tracer = otel.Tracer("ucms/internal/application/staff/cmd")
	logger = otelslog.NewLogger("ucms/internal/application/staff/cmd")
)

var (
	ErrEmailNotAvailable    = errorx.NewDuplicateEntry().WithKey(i18nx.KeyEmailNotAvailable)
	ErrBarcodeNotAvailable  = errorx.NewDuplicateEntry().WithKey(i18nx.KeyBarcodeNotAvailable)
	ErrUsernameNotAvailable = errorx.NewDuplicateEntry().WithKey(i18nx.KeyUsernameNotAvailable)
)

type StaffInvitationRepo interface {
	SaveStaffInvitation(ctx context.Context, invitation *staffinvitation.StaffInvitation) error
	UpdateStaffInvitation(ctx context.Context, id staffinvitation.ID, fn func(context.Context, *staffinvitation.StaffInvitation) error) error
	GetStaffInvitationByCode(ctx context.Context, code string) (*staffinvitation.StaffInvitation, error)
}

type StaffRepo interface {
	IsStaffExists(
		ctx context.Context,
		email string,
		username string,
		barcode user.Barcode,
	) (emailExists bool, usernameExists bool, barcodeExists bool, err error)
	SaveStaff(ctx context.Context, staff *user.Staff) error
}

type CreateInvitation struct {
	CreatorID       user.ID
	RecipientsEmail []string
	ValidFrom       *time.Time
	ValidUntil      *time.Time
}

type CreateInvitationHandler struct {
	tracer trace.Tracer
	logger *slog.Logger
	repo   StaffInvitationRepo
}

type CreateInvitationHandlerArgs struct {
	Tracer              trace.Tracer
	Logger              *slog.Logger
	StaffInvitationRepo StaffInvitationRepo
}

func NewCreateInvitationHandler(args CreateInvitationHandlerArgs) *CreateInvitationHandler {
	h := &CreateInvitationHandler{
		tracer: args.Tracer,
		logger: args.Logger,
		repo:   args.StaffInvitationRepo,
	}

	if h.tracer == nil {
		h.tracer = tracer
	}
	if h.logger == nil {
		h.logger = logger
	}

	return h
}

func (h *CreateInvitationHandler) Handle(ctx context.Context, cmd CreateInvitation) error {
	const op = "cmd.CreateInvitationHandler.Handle"
	ctx, span := h.tracer.Start(ctx, "CreateInvitationHandler.Handle", trace.WithAttributes(
		attribute.String("creator_id", cmd.CreatorID.String()),
		attribute.Int("recipients_count", len(cmd.RecipientsEmail)),
	))
	defer span.End()

	invitation, err := staffinvitation.NewStaffInvitation(staffinvitation.CreateArgs{
		RecipientsEmail: cmd.RecipientsEmail,
		CreatorID:       cmd.CreatorID,
		ValidFrom:       cmd.ValidFrom,
		ValidUntil:      cmd.ValidUntil,
	})
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to create new staff invitation")
		return errorx.Wrap(err, op)
	}

	err = h.repo.SaveStaffInvitation(ctx, invitation)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to save staff invitation")
		return errorx.Wrap(err, op)
	}

	return nil
}

type UpdateInvitationRecipients struct {
	CreatorID       user.ID
	InvitationID    staffinvitation.ID
	RecipientsEmail []string
}

type UpdateInvitationRecipientsHandler struct {
	tracer trace.Tracer
	logger *slog.Logger
	repo   StaffInvitationRepo
}

type UpdateInvitationRecipientsHandlerArgs struct {
	Tracer              trace.Tracer
	Logger              *slog.Logger
	StaffInvitationRepo StaffInvitationRepo
}

func NewUpdateInvitationRecipientsHandler(args UpdateInvitationRecipientsHandlerArgs) *UpdateInvitationRecipientsHandler {
	h := &UpdateInvitationRecipientsHandler{
		tracer: args.Tracer,
		logger: args.Logger,
		repo:   args.StaffInvitationRepo,
	}

	if h.tracer == nil {
		h.tracer = tracer
	}
	if h.logger == nil {
		h.logger = logger
	}

	return h
}

func (h *UpdateInvitationRecipientsHandler) Handle(ctx context.Context, cmd UpdateInvitationRecipients) error {
	const op = "cmd.UpdateInvitationRecipientsHandler.Handle"
	ctx, span := h.tracer.Start(ctx, "UpdateInvitationRecipientsHandler.Handle", trace.WithAttributes(
		attribute.String("invitation_id", cmd.InvitationID.String()),
		attribute.String("creator_id", cmd.CreatorID.String()),
		attribute.Int("recipients_count", len(cmd.RecipientsEmail)),
	))
	defer span.End()

	err := h.repo.UpdateStaffInvitation(ctx, cmd.InvitationID, func(ctx context.Context, si *staffinvitation.StaffInvitation) error {
		if err := si.UpdateRecipients(cmd.CreatorID, cmd.RecipientsEmail); err != nil {
			trace.SpanFromContext(ctx).AddEvent("failed to update recipients")
			return err
		}

		return nil
	})
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to update staff invitation")
		return errorx.Wrap(err, op)
	}

	return nil
}

type UpdateInvitationValidity struct {
	CreatorID    user.ID
	InvitationID staffinvitation.ID
	ValidFrom    *time.Time
	ValidUntil   *time.Time
}

type UpdateInvitationValidityHandler struct {
	tracer trace.Tracer
	logger *slog.Logger
	repo   StaffInvitationRepo
}

type UpdateInvitationValidityHandlerArgs struct {
	Tracer              trace.Tracer
	Logger              *slog.Logger
	StaffInvitationRepo StaffInvitationRepo
}

func NewUpdateInvitationValidityHandler(args UpdateInvitationValidityHandlerArgs) *UpdateInvitationValidityHandler {
	h := &UpdateInvitationValidityHandler{
		tracer: args.Tracer,
		logger: args.Logger,
		repo:   args.StaffInvitationRepo,
	}

	if h.tracer == nil {
		h.tracer = tracer
	}
	if h.logger == nil {
		h.logger = logger
	}

	return h
}

func (h *UpdateInvitationValidityHandler) Handle(ctx context.Context, cmd UpdateInvitationValidity) error {
	const op = "cmd.UpdateInvitationValidityHandler.Handle"
	ctx, span := h.tracer.Start(ctx, "UpdateInvitationValidityHandler.Handle", trace.WithAttributes(
		attribute.String("invitation_id", cmd.InvitationID.String()),
		attribute.String("creator_id", cmd.CreatorID.String()),
	))
	defer span.End()

	err := h.repo.UpdateStaffInvitation(ctx, cmd.InvitationID, func(ctx context.Context, si *staffinvitation.StaffInvitation) error {
		if err := si.UpdateValidity(cmd.CreatorID, cmd.ValidFrom, cmd.ValidUntil); err != nil {
			trace.SpanFromContext(ctx).AddEvent("failed to update validity period")
			return err
		}

		return nil
	})
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to update staff invitation validity")
		return errorx.Wrap(err, op)
	}

	return nil
}

type DeleteInvitation struct {
	CreatorID    user.ID
	InvitationID staffinvitation.ID
}

type DeleteInvitationHandler struct {
	tracer trace.Tracer
	logger *slog.Logger
	repo   StaffInvitationRepo
}

type DeleteInvitationHandlerArgs struct {
	Tracer              trace.Tracer
	Logger              *slog.Logger
	StaffInvitationRepo StaffInvitationRepo
}

func NewDeleteInvitationHandler(args DeleteInvitationHandlerArgs) *DeleteInvitationHandler {
	h := &DeleteInvitationHandler{
		tracer: args.Tracer,
		logger: args.Logger,
		repo:   args.StaffInvitationRepo,
	}

	if h.tracer == nil {
		h.tracer = tracer
	}
	if h.logger == nil {
		h.logger = logger
	}

	return h
}

func (h *DeleteInvitationHandler) Handle(ctx context.Context, cmd DeleteInvitation) error {
	const op = "cmd.DeleteInvitationHandler.Handle"
	ctx, span := h.tracer.Start(ctx, "DeleteInvitationHandler.Handle", trace.WithAttributes(
		attribute.String("invitation_id", cmd.InvitationID.String()),
		attribute.String("creator_id", cmd.CreatorID.String()),
	))
	defer span.End()

	err := h.repo.UpdateStaffInvitation(ctx, cmd.InvitationID, func(ctx context.Context, si *staffinvitation.StaffInvitation) error {
		if err := si.MarkDeleted(cmd.CreatorID); err != nil {
			trace.SpanFromContext(ctx).AddEvent("failed to mark invitation as deleted")
			return err
		}

		return nil
	})
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to delete staff invitation")
		return errorx.Wrap(err, op)
	}

	return nil
}

type ValidateInvitation struct {
	InvitationCode string
	Email          string
}

type ValidateInvitationHandler struct {
	tracer trace.Tracer
	logger *slog.Logger
	repo   StaffInvitationRepo
}

type ValidateInvitationHandlerArgs struct {
	Tracer              trace.Tracer
	Logger              *slog.Logger
	StaffInvitationRepo StaffInvitationRepo
}

func NewValidateInvitationHandler(args ValidateInvitationHandlerArgs) *ValidateInvitationHandler {
	h := &ValidateInvitationHandler{
		tracer: args.Tracer,
		logger: args.Logger,
		repo:   args.StaffInvitationRepo,
	}

	if h.tracer == nil {
		h.tracer = tracer
	}
	if h.logger == nil {
		h.logger = logger
	}

	return h
}

func (h *ValidateInvitationHandler) Handle(ctx context.Context, cmd ValidateInvitation) error {
	const op = "cmd.ValidateInvitationHandler.Handle"
	ctx, span := h.tracer.Start(ctx, "ValidateInvitationHandler.Handle", trace.WithAttributes(
		attribute.String("invitation_code", cmd.InvitationCode),
		attribute.String("email", cmd.Email),
	))
	defer span.End()

	invitation, err := h.repo.GetStaffInvitationByCode(ctx, cmd.InvitationCode)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to get staff invitation by code")
		if errorx.IsNotFound(err) {
			return staffinvitation.ErrNotFoundOrDeleted.WithCause(err, op)
		}
		return errorx.Wrap(err, op)
	}

	if err := invitation.ValidateInvitationAccess(cmd.Email, cmd.InvitationCode); err != nil {
		otelx.RecordSpanError(span, err, "invitation validation failed")
		return errorx.Wrap(err, op)
	}

	return nil
}

type AcceptInvitation struct {
	InvitationCode string
	Email          string
	Barcode        user.Barcode
	Username       string
	Password       string
	FirstName      string
	LastName       string
}

type AcceptInvitationHandler struct {
	tracer    trace.Tracer
	logger    *slog.Logger
	repo      StaffInvitationRepo
	staffRepo StaffRepo
}

type AcceptInvitationHandlerArgs struct {
	Tracer              trace.Tracer
	Logger              *slog.Logger
	StaffInvitationRepo StaffInvitationRepo
	StaffRepo           StaffRepo
}

func NewAcceptInvitationHandler(args AcceptInvitationHandlerArgs) *AcceptInvitationHandler {
	h := &AcceptInvitationHandler{
		tracer:    args.Tracer,
		logger:    args.Logger,
		repo:      args.StaffInvitationRepo,
		staffRepo: args.StaffRepo,
	}

	if h.tracer == nil {
		h.tracer = tracer
	}
	if h.logger == nil {
		h.logger = logger
	}

	return h
}

func (h *AcceptInvitationHandler) Handle(ctx context.Context, cmd AcceptInvitation) error {
	const op = "cmd.AcceptInvitationHandler.Handle"
	ctx, span := h.tracer.Start(ctx, "AcceptInvitationHandler.Handle", trace.WithAttributes(
		attribute.String("invitation_code", cmd.InvitationCode),
		attribute.String("email", cmd.Email),
		attribute.String("barcode", cmd.Barcode.String()),
		attribute.String("username", cmd.Username),
	))
	defer span.End()

	invitation, err := h.repo.GetStaffInvitationByCode(ctx, cmd.InvitationCode)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to get staff invitation by code")
		if errorx.IsNotFound(err) {
			return staffinvitation.ErrNotFoundOrDeleted.WithCause(err, op)
		}
		return errorx.Wrap(err, op)
	}

	if err := invitation.ValidateInvitationAccess(cmd.Email, cmd.InvitationCode); err != nil {
		otelx.RecordSpanError(span, err, "invitation validation failed")
		return errorx.Wrap(err, op)
	}

	emailExists, usernameExists, barcodeExists, err := h.staffRepo.IsStaffExists(ctx, cmd.Email, cmd.Username, cmd.Barcode)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to check if staff exists")
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

	staff, err := user.AcceptStaffInvitation(user.AcceptStaffInvitationArgs{
		Email:        cmd.Email,
		Barcode:      cmd.Barcode,
		Username:     cmd.Username,
		Password:     cmd.Password,
		FirstName:    cmd.FirstName,
		LastName:     cmd.LastName,
		InvitationID: uuid.UUID(invitation.ID()),
	})
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to create staff")
		return errorx.Wrap(err, op)
	}

	err = h.staffRepo.SaveStaff(ctx, staff)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to save staff")
		return errorx.Wrap(err, op)
	}

	return nil
}
