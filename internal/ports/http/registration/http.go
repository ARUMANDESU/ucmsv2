package registrationhttp

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/oapi-codegen/runtime/types"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	registrationapp "github.com/ARUMANDESU/ucms/internal/application/registration"
	"github.com/ARUMANDESU/ucms/internal/application/registration/cmd"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/pkg/httpx"
	"github.com/ARUMANDESU/ucms/pkg/sanitizex"
	"github.com/ARUMANDESU/ucms/pkg/validationx"
)

var (
	tracer = otel.Tracer("ucms/internal/ports/http/registration")
	logger = otelslog.NewLogger("ucms/internal/ports/http/registration")
)

type HTTP struct {
	tracer     trace.Tracer
	logger     *slog.Logger
	cmd        *registrationapp.Command
	errhandler *httpx.ErrorHandler
}

type Args struct {
	Tracer  trace.Tracer
	Logger  *slog.Logger
	Command *registrationapp.Command
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
		cmd:        args.Command,
		errhandler: httpx.NewErrorHandler(),
	}
}

func (h *HTTP) Route(r chi.Router) {
	r.Post("/v1/registrations/verify", h.Verify)
	r.Post("/v1/registrations/students/start", h.StartStudentRegistration)
	r.Post("/v1/registrations/students/complete", h.CompleteStudentRegistration)
	r.Post("/v1/registrations/resend", h.ResendVerificationCode)
}

func (h *HTTP) StartStudentRegistration(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "StartStudentRegistration")
	defer span.End()

	var req PostV1RegistrationsStudentsStartJSONRequestBody
	if err := httpx.ReadJSON(w, r, &req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read json")
		httpx.BadRequest(w, r, err.Error())
		return
	}

	req.Email = types.Email(sanitizex.CleanSingleLine(string(req.Email)))

	err := validation.ValidateStruct(&req,
		validation.Field(&req.Email, validationx.EmailRules...),
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to validate request body")
		h.errhandler.HandleError(w, r, err)
		return
	}

	if err := h.cmd.StartStudent.Handle(ctx, cmd.StartStudent{Email: string(req.Email)}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to start studen registration")
		h.errhandler.HandleError(w, r, err)
		return
	}

	httpx.Success(w, r, http.StatusAccepted, nil)
}

func (h *HTTP) Verify(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "VerifyRegistration")
	defer span.End()

	var req PostV1RegistrationsVerifyJSONRequestBody
	if err := httpx.ReadJSON(w, r, &req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read json")
		httpx.BadRequest(w, r, err.Error())
		return
	}

	req.Email = types.Email(sanitizex.CleanSingleLine(string(req.Email)))
	req.VerificationCode = sanitizex.CleanSingleLine(req.VerificationCode)

	err := validation.ValidateStruct(
		&req,
		validation.Field(&req.Email, validationx.EmailRules...),
		validation.Field(&req.VerificationCode,
			validation.Required,
			validation.Length(registration.VerificationCodeLength, registration.VerificationCodeLength),
			is.Alphanumeric,
		),
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to validate request body")
		h.errhandler.HandleError(w, r, err)
		return
	}

	cmd := cmd.Verify{
		Email: string(req.Email),
		Code:  string(req.VerificationCode),
	}
	if err := h.cmd.Verify.Handle(ctx, cmd); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to verify registration email")
		h.errhandler.HandleError(w, r, err)
		return
	}

	httpx.Success(w, r, http.StatusOK, nil)
}

func (h *HTTP) CompleteStudentRegistration(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "CompleteStudentRegistration")
	defer span.End()

	var req PostV1RegistrationsStudentsCompleteJSONRequestBody
	if err := httpx.ReadJSON(w, r, &req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read json")
		httpx.BadRequest(w, r, err.Error())
		return
	}

	req.Email = types.Email(sanitizex.CleanSingleLine(string(req.Email)))
	req.Barcode = Barcode(sanitizex.CleanSingleLine(string(req.Barcode)))
	req.VerificationCode = sanitizex.CleanSingleLine(req.VerificationCode)
	req.FirstName = sanitizex.CleanSingleLine(req.FirstName)
	req.LastName = sanitizex.CleanSingleLine(req.LastName)
	req.Password = strings.TrimSpace(req.Password)

	err := validation.ValidateStruct(&req,
		validation.Field(&req.Email, validationx.EmailRules...),
		validation.Field(&req.VerificationCode,
			validation.Required,
			validation.Length(registration.VerificationCodeLength, registration.VerificationCodeLength),
			is.Alphanumeric,
		),
		validation.Field(&req.FirstName, validationx.NameRules...),
		validation.Field(&req.LastName, validationx.NameRules...),
		validation.Field(&req.Password, validationx.PasswordRules...),
		validation.Field(&req.Barcode, validation.Required, validation.Length(1, 100), is.Alphanumeric),
		validation.Field(&req.GroupId, validationx.Required),
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to validate request body")
		h.errhandler.HandleError(w, r, err)
		return
	}

	cmd := cmd.StudentComplete{
		Email:            string(req.Email),
		VerificationCode: string(req.VerificationCode),
		Barcode:          string(req.Barcode),
		FirstName:        string(req.FirstName),
		LastName:         string(req.LastName),
		Password:         string(req.Password),
		GroupID:          req.GroupId,
	}
	if err := h.cmd.StudentComplete.Handle(ctx, cmd); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to complete student registration")
		h.errhandler.HandleError(w, r, err)
		return
	}

	httpx.Success(w, r, http.StatusOK, nil)
}

func (h *HTTP) ResendVerificationCode(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "ResendVerificationCode")
	defer span.End()

	var req PostV1RegistrationsResendJSONRequestBody
	if err := httpx.ReadJSON(w, r, &req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read json")
		httpx.BadRequest(w, r, err.Error())
		return
	}

	req.Email = types.Email(sanitizex.CleanSingleLine(string(req.Email)))

	err := validation.ValidateStruct(&req,
		validation.Field(&req.Email, validationx.EmailRules...),
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to validate request body")
		h.errhandler.HandleError(w, r, err)
		return
	}

	cmd := cmd.ResendCode{Email: string(req.Email)}
	if err := h.cmd.ResendCode.Handle(ctx, cmd); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to resend registration email verification code")
		h.errhandler.HandleError(w, r, err)
		return
	}

	httpx.Success(w, r, http.StatusAccepted, nil)
}
