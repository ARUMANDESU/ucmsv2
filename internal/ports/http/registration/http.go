package registrationhttp

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	registrationapp "gitlab.com/ucmsv2/ucms-backend/internal/application/registration"
	"gitlab.com/ucmsv2/ucms-backend/internal/application/registration/cmd"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/group"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/registration"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/pkg/env"
	"gitlab.com/ucmsv2/ucms-backend/pkg/httpx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/logging"
	"gitlab.com/ucmsv2/ucms-backend/pkg/otelx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/sanitizex"
	"gitlab.com/ucmsv2/ucms-backend/pkg/validationx"
)

var (
	tracer = otel.Tracer("ucms/internal/ports/http/registration")
	logger = otelslog.NewLogger("ucms/internal/ports/http/registration")
)

type HTTP struct {
	tracer     trace.Tracer
	logger     *slog.Logger
	cmd        *registrationapp.Command
	query      *registrationapp.Query
	errhandler *httpx.ErrorHandler
}

type Args struct {
	Tracer     trace.Tracer
	Logger     *slog.Logger
	App        *registrationapp.App
	Errhandler *httpx.ErrorHandler
}

func NewHTTP(args Args) *HTTP {
	if args.Tracer == nil {
		args.Tracer = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}

	return &HTTP{
		tracer:     args.Tracer,
		logger:     args.Logger,
		cmd:        &args.App.Command,
		query:      &args.App.Query,
		errhandler: args.Errhandler,
	}
}

func (h *HTTP) Route(r chi.Router) {
	r.Route("/v1/registrations", func(r chi.Router) {
		r.Post("/verify", h.Verify)
		r.Post("/resend", h.ResendVerificationCode)
		r.Post("/students/start", h.StartStudentRegistration)
		r.Post("/students/complete", h.CompleteStudentRegistration)
	})

	if env.Current() == env.Dev || env.Current() == env.Local || env.Current() == env.Test {
		r.Get("/dev/registrations/verification-code/{email}", h.GetVerificationCode)
	}
}

type StartStudentRegistrationRequest struct {
	Email string `json:"email"`
}

func (r *StartStudentRegistrationRequest) Sanitized() {
	r.Email = sanitizex.CleanSingleLine(r.Email)
}

func (r *StartStudentRegistrationRequest) SetSpanAttrs(span trace.Span) {
	otelx.SetSpanAttrs(span, map[string]any{"email": logging.RedactEmail(r.Email)})
}

func (r *StartStudentRegistrationRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Email, validationx.EmailRules...),
	)
}

func (h *HTTP) StartStudentRegistration(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "StartStudentRegistration")
	defer span.End()

	var req StartStudentRegistrationRequest
	if err := httpx.ReadJSON(w, r, &req); err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to read json")
		return
	}

	req.Sanitized()
	req.SetSpanAttrs(span)
	err := req.Validate()
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to validate request body")
		return
	}

	if err := h.cmd.StartStudent.Handle(ctx, cmd.StartStudent{Email: req.Email}); err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to start student registration")
		return
	}

	httpx.Success(w, r, http.StatusAccepted, nil)
}

type VerifyRequest struct {
	Email            string `json:"email"`
	VerificationCode string `json:"verification_code"`
}

func (r *VerifyRequest) Sanitized() {
	r.Email = sanitizex.CleanSingleLine(r.Email)
	r.VerificationCode = sanitizex.CleanSingleLine(r.VerificationCode)
}

func (r *VerifyRequest) SetSpanAttrs(span trace.Span) {
	otelx.SetSpanAttrs(span, map[string]any{"email": logging.RedactEmail(r.Email)})
}

func (r *VerifyRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Email, validationx.EmailRules...),
		validation.Field(&r.VerificationCode,
			validation.Required,
			validation.Length(registration.VerificationCodeLength, registration.VerificationCodeLength),
			is.Alphanumeric,
		),
	)
}

func (h *HTTP) Verify(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "VerifyRegistration")
	defer span.End()

	var req VerifyRequest
	if err := httpx.ReadJSON(w, r, &req); err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to read json")
		return
	}

	req.Sanitized()
	req.SetSpanAttrs(span)
	err := req.Validate()
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to validate request body")
		return
	}

	cmd := cmd.Verify{
		Email: req.Email,
		Code:  req.VerificationCode,
	}
	if err := h.cmd.Verify.Handle(ctx, cmd); err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to verify registration")
		return
	}

	httpx.Success(w, r, http.StatusOK, nil)
}

type CompleteStudentRegistrationRequest struct {
	Barcode          string    `json:"barcode"`
	Username         string    `json:"username"`
	Email            string    `json:"email"`
	FirstName        string    `json:"first_name"`
	GroupId          uuid.UUID `json:"group_id"`
	LastName         string    `json:"last_name"`
	Password         string    `json:"password"`
	VerificationCode string    `json:"verification_code"`
}

func (r *CompleteStudentRegistrationRequest) Sanitized() {
	r.Barcode = sanitizex.CleanSingleLine(r.Barcode)
	r.Username = sanitizex.CleanSingleLine(r.Username)
	r.Email = sanitizex.CleanSingleLine(r.Email)
	r.FirstName = sanitizex.CleanSingleLine(r.FirstName)
	r.LastName = sanitizex.CleanSingleLine(r.LastName)
	r.VerificationCode = sanitizex.CleanSingleLine(r.VerificationCode)
	r.Password = strings.TrimSpace(r.Password)
}

func (r *CompleteStudentRegistrationRequest) SetSpanAttrs(span trace.Span) {
	otelx.SetSpanAttrs(span, map[string]any{
		"email":    logging.RedactEmail(r.Email),
		"username": logging.RedactUsername(r.Username),
		"group_id": r.GroupId.String(),
	})
}

func (r *CompleteStudentRegistrationRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Email, validationx.EmailRules...),
		validation.Field(&r.VerificationCode,
			validation.Required,
			validation.Length(registration.VerificationCodeLength, registration.VerificationCodeLength),
			is.Alphanumeric,
		),
		validation.Field(&r.Username, validation.Required, validation.Length(2, 100)),
		validation.Field(&r.FirstName, validationx.NameRules...),
		validation.Field(&r.LastName, validationx.NameRules...),
		validation.Field(&r.Password, validationx.PasswordRules...),
		validation.Field(&r.Barcode, validation.Required, validation.Length(1, 100), is.Alphanumeric),
		validation.Field(&r.GroupId, validationx.Required),
	)
}

func (h *HTTP) CompleteStudentRegistration(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "CompleteStudentRegistration")
	defer span.End()

	var req CompleteStudentRegistrationRequest
	if err := httpx.ReadJSON(w, r, &req); err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to read json")
		return
	}

	req.Sanitized()
	req.SetSpanAttrs(span)
	err := req.Validate()
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to validate request body")
		return
	}

	cmd := cmd.StudentComplete{
		Email:            req.Email,
		VerificationCode: req.VerificationCode,
		Barcode:          user.Barcode(req.Barcode),
		Username:         req.Username,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		Password:         req.Password,
		GroupID:          group.ID(req.GroupId),
	}
	if err := h.cmd.StudentComplete.Handle(ctx, cmd); err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to complete student registration")
		return
	}

	httpx.Success(w, r, http.StatusOK, nil)
}

type ResendVerificationCodeRequest struct {
	Email string `json:"email"`
}

func (r *ResendVerificationCodeRequest) Sanitized() {
	r.Email = sanitizex.CleanSingleLine(r.Email)
}

func (r *ResendVerificationCodeRequest) SetSpanAttrs(span trace.Span) {
	otelx.SetSpanAttrs(span, map[string]any{"email": logging.RedactEmail(r.Email)})
}

func (r *ResendVerificationCodeRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Email, validationx.EmailRules...),
	)
}

func (h *HTTP) ResendVerificationCode(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "ResendVerificationCode")
	defer span.End()

	var req ResendVerificationCodeRequest
	if err := httpx.ReadJSON(w, r, &req); err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to read json")
		return
	}

	req.Sanitized()
	req.SetSpanAttrs(span)
	err := req.Validate()
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to validate request body")
		return
	}

	cmd := cmd.ResendCode{Email: req.Email}
	if err := h.cmd.ResendCode.Handle(ctx, cmd); err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to resend verification code")
		return
	}

	httpx.Success(w, r, http.StatusAccepted, nil)
}

func (h *HTTP) GetVerificationCode(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "GetVerificationCode")
	defer span.End()

	email := chi.URLParam(r, "email")
	email = sanitizex.CleanSingleLine(email)

	err := validation.Validate(email, validationx.EmailRules...)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to validate email")
		return
	}

	code, err := h.query.GetVerificationCode.Handle(ctx, email)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to get verification code")
		return
	}

	httpx.Success(w, r, http.StatusOK, httpx.Envelope{"verification_code": code})
}
