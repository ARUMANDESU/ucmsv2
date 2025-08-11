package registrationhttp

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/application/registration"
	"github.com/ARUMANDESU/ucms/internal/application/registration/cmd"
	"github.com/ARUMANDESU/ucms/pkg/httpx"
)

var (
	tracer = otel.Tracer("ucms/internal/ports/http/registration")
	logger = otelslog.NewLogger("ucms/internal/ports/http/registration")
)

type HTTP struct {
	tracer trace.Tracer
	logger *slog.Logger
	cmd    registration.Command
}

type Args struct {
	Tracer  trace.Tracer
	Logger  *slog.Logger
	Command registration.Command
}

func NewHTTP(args Args) *HTTP {
	if args.Tracer == nil {
		args.Tracer = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}

	return &HTTP{
		tracer: args.Tracer,
		logger: args.Logger,
		cmd:    args.Command,
	}
}

func (h *HTTP) Route(r chi.Router) {
	r.Post("/v1/registrations/verify", h.Verify)
	r.Post("/v1/registrations/students/start", h.StartStudentRegistration)
	r.Post("/v1/registrations/students/complete", h.CompleteStudentRegistration)
}

func (h *HTTP) StartStudentRegistration(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "StartStudentRegistration")
	defer span.End()

	var body PostV1RegistrationsStudentsStartJSONRequestBody
	if err := httpx.ReadJSON(w, r, &body); err != nil {
		httpx.BadRequest(w, r, err.Error())
		return
	}

	cmd := cmd.StartStudent{Email: string(body.Email)}
	if err := h.cmd.StartStudent.Handle(ctx, cmd); err != nil {
		httpx.HandleError(w, r, err)
		return
	}

	httpx.Success(w, r, http.StatusAccepted, nil)
}

func (h *HTTP) Verify(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "VerifyRegistration")
	defer span.End()

	var body PostV1RegistrationsVerifyJSONRequestBody
	if err := httpx.ReadJSON(w, r, &body); err != nil {
		httpx.BadRequest(w, r, err.Error())
		return
	}

	cmd := cmd.Verify{
		Email: string(body.Email),
		Code:  string(body.VerificationCode),
	}
	if err := h.cmd.Verify.Handle(ctx, cmd); err != nil {
		httpx.HandleError(w, r, err)
		return
	}

	httpx.Success(w, r, http.StatusOK, nil)
}

func (h *HTTP) CompleteStudentRegistration(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "CompleteStudentRegistration")
	defer span.End()

	var body PostV1RegistrationsStudentsCompleteJSONRequestBody
	if err := httpx.ReadJSON(w, r, &body); err != nil {
		httpx.BadRequest(w, r, err.Error())
		return
	}

	cmd := cmd.StudentComplete{
		Email:            string(body.Email),
		VerificationCode: string(body.VerificationCode),
		Barcode:          string(body.Barcode),
		FirstName:        string(body.FirstName),
		LastName:         string(body.LastName),
		Password:         string(body.Password),
		GroupID:          body.GroupId,
	}
	if err := h.cmd.StudentComplete.Handle(ctx, cmd); err != nil {
		httpx.HandleError(w, r, err)
		return
	}

	httpx.Success(w, r, http.StatusOK, nil)
}
