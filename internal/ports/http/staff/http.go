package staff

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
	"github.com/go-chi/chi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	staffapp "github.com/ARUMANDESU/ucms/internal/application/staff"
	"github.com/ARUMANDESU/ucms/internal/application/staff/cmd"
	"github.com/ARUMANDESU/ucms/internal/domain/staffinvitation"
	"github.com/ARUMANDESU/ucms/pkg/ctxs"
	"github.com/ARUMANDESU/ucms/pkg/httpx"
	"github.com/ARUMANDESU/ucms/pkg/otelx"
	"github.com/ARUMANDESU/ucms/pkg/sanitizex"
)

var (
	tracer = otel.Tracer("ucms/internal/ports/http/staff")
	logger = otelslog.NewLogger("ucms/internal/ports/http/staff")
)

var (
	recipientsEmailRules = []validation.Rule{validation.Count(0, 100), validation.Each(validation.Required, is.Email)}
	validityRules        = []validation.Rule{validation.NilOrNotEmpty}
)

type HTTP struct {
	tracer     trace.Tracer
	logger     *slog.Logger
	cmd        *staffapp.Command
	query      *staffapp.Query
	errhandler *httpx.ErrorHandler
}

type Args struct {
	Tracer     trace.Tracer
	Logger     *slog.Logger
	App        *staffapp.App
	Errhandler *httpx.ErrorHandler
}

func NewHTTP(args Args) *HTTP {
	h := &HTTP{
		tracer:     args.Tracer,
		logger:     args.Logger,
		cmd:        &args.App.Command,
		query:      &args.App.Query,
		errhandler: args.Errhandler,
	}

	if h.tracer == nil {
		h.tracer = tracer
	}
	if h.logger == nil {
		h.logger = logger
	}
	if h.errhandler == nil {
		h.errhandler = httpx.NewErrorHandler()
	}

	return h
}

func (h *HTTP) Route(r chi.Router) {
	r.Route("/v1/staffs", func(r chi.Router) {
		r.Route("/invitations", func(r chi.Router) {
			r.Post("/", h.CreateInvitation)
			r.Put("/{invitation_id}/recipients", h.UpdateInvitationRecipients)
			r.Put("/{invitation_id}/validity", h.UpdateInvitationValidity)
			r.Delete("/{invitation_id}", h.DeleteInvitation)
		})
	})
}

type CreateInvitationRequest struct {
	Recipients []string   `json:"recipients_email"`
	ValidFrom  *time.Time `json:"valid_from"`
	ValidUntil *time.Time `json:"valid_until"`
}

func (c *CreateInvitationRequest) Sanitize() {
	c.Recipients = sanitizex.DeduplicateSlice(c.Recipients, sanitizex.StringTransformFunc(sanitizex.CleanSingleLine))
}

func (c *CreateInvitationRequest) SetSpanAttrs(span trace.Span) {
	otelx.SetSpanAttrs(span, map[string]any{
		"request.recipients_count": len(c.Recipients),
		"request.valid_from":       c.ValidFrom,
		"request.valid_until":      c.ValidUntil,
	})
}

func (c *CreateInvitationRequest) Validate() error {
	return validation.ValidateStruct(c,
		validation.Field(&c.Recipients, recipientsEmailRules...),
		validation.Field(&c.ValidFrom, validityRules...),
		validation.Field(&c.ValidUntil, validityRules...),
	)
}

func (h *HTTP) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "HTTP.CreateInvitation")
	defer span.End()

	ctxUser, err := ctxs.UserFromCtx(ctx)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to get user from context")
		return
	}
	ctxUser.SetSpanAttrs(span)

	var req CreateInvitationRequest
	if err := httpx.ReadJSON(w, r, &req); err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to read body")
		return
	}

	req.Sanitize()
	req.SetSpanAttrs(span)
	err = req.Validate()
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "validation failed")
		return
	}

	err = h.cmd.CreateInvitation.Handle(ctx, cmd.CreateInvitation{
		CreatorID:       ctxUser.ID,
		RecipientsEmail: req.Recipients,
		ValidFrom:       req.ValidFrom,
		ValidUntil:      req.ValidUntil,
	})
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to create invitation")
		return
	}

	httpx.Success(w, r, http.StatusCreated, nil)
}

type UpdateInvitationRecipientsRequest struct {
	Recipients []string `json:"recipients_email"`
}

func (r *UpdateInvitationRecipientsRequest) Sanitize() {
	r.Recipients = sanitizex.DeduplicateSlice(r.Recipients, sanitizex.StringTransformFunc(sanitizex.CleanSingleLine))
}

func (r *UpdateInvitationRecipientsRequest) SetSpanAttrs(span trace.Span) {
	otelx.SetSpanAttrs(span, map[string]any{"request.recipients_count": len(r.Recipients)})
}

func (r *UpdateInvitationRecipientsRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Recipients, recipientsEmailRules...),
	)
}

func (h *HTTP) UpdateInvitationRecipients(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "HTTP.UpdateInvitationRecipients")
	defer span.End()

	ctxUser, err := ctxs.UserFromCtx(ctx)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to get user from context")
		return
	}
	ctxUser.SetSpanAttrs(span)

	invitationID, err := httpx.ReadUUIDUrlParam(r, "invitation_id")
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "invalid invitation_id")
		return
	}
	span.SetAttributes(attribute.String("request.invitation_id", invitationID.String()))

	var req UpdateInvitationRecipientsRequest
	if err := httpx.ReadJSON(w, r, &req); err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to read body")
		return
	}

	req.Sanitize()
	req.SetSpanAttrs(span)
	err = req.Validate()
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "validation failed")
		return
	}

	err = h.cmd.UpdateInvitationRecipients.Handle(ctx, cmd.UpdateInvitationRecipients{
		InvitationID:    staffinvitation.ID(invitationID),
		CreatorID:       ctxUser.ID,
		RecipientsEmail: req.Recipients,
	})
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to update invitation recipients")
		return
	}

	httpx.Success(w, r, http.StatusNoContent, nil)
}

type UpdateInvitationValidityRequest struct {
	ValidFrom  *time.Time `json:"valid_from"`
	ValidUntil *time.Time `json:"valid_until"`
}

func (r *UpdateInvitationValidityRequest) SetSpanAttrs(span trace.Span) {
	otelx.SetSpanAttrs(span, map[string]any{
		"request.valid_from":  r.ValidFrom,
		"request.valid_until": r.ValidUntil,
	})
}

func (r *UpdateInvitationValidityRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.ValidFrom, validityRules...),
		validation.Field(&r.ValidUntil, validityRules...),
	)
}

func (h *HTTP) UpdateInvitationValidity(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "HTTP.UpdateInvitationValidity")
	defer span.End()

	ctxUser, err := ctxs.UserFromCtx(ctx)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to get user from context")
		return
	}
	ctxUser.SetSpanAttrs(span)

	invitationID, err := httpx.ReadUUIDUrlParam(r, "invitation_id")
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "invalid invitation_id")
		return
	}
	span.SetAttributes(attribute.String("request.invitation_id", invitationID.String()))

	var req UpdateInvitationValidityRequest
	if err := httpx.ReadJSON(w, r, &req); err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to read body")
		return
	}

	req.SetSpanAttrs(span)
	err = req.Validate()
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "validation failed")
		return
	}

	err = h.cmd.UpdateInvitationValidity.Handle(ctx, cmd.UpdateInvitationValidity{
		InvitationID: staffinvitation.ID(invitationID),
		CreatorID:    ctxUser.ID,
		ValidFrom:    req.ValidFrom,
		ValidUntil:   req.ValidUntil,
	})
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to update invitation validity")
		return
	}

	httpx.Success(w, r, http.StatusNoContent, nil)
}

func (h *HTTP) DeleteInvitation(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "HTTP.DeleteInvitation")
	defer span.End()

	ctxUser, err := ctxs.UserFromCtx(ctx)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to get user from context")
		return
	}
	ctxUser.SetSpanAttrs(span)

	invitationID, err := httpx.ReadUUIDUrlParam(r, "invitation_id")
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "invalid invitation_id")
		return
	}
	span.SetAttributes(attribute.String("request.invitation_id", invitationID.String()))

	err = h.cmd.DeleteInvitation.Handle(ctx, cmd.DeleteInvitation{
		InvitationID: staffinvitation.ID(invitationID),
		CreatorID:    ctxUser.ID,
	})
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to delete invitation")
		return
	}

	httpx.Success(w, r, http.StatusNoContent, nil)
}
