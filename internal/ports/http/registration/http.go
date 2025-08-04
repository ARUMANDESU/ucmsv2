package registrationhttp

import (
	"net/http"

	"github.com/go-chi/chi"

	"github.com/ARUMANDESU/ucms/internal/application/registration"
	"github.com/ARUMANDESU/ucms/internal/application/registration/cmd"
	"github.com/ARUMANDESU/ucms/pkg/httpx"
)

type HTTP struct {
	cmd registration.Command
}

type Args struct {
	Command registration.Command
}

func NewHTTP(args Args) *HTTP {
	return &HTTP{
		cmd: args.Command,
	}
}

func (h *HTTP) Route(r chi.Router) {
	r.Post("/v1/registration/start/student", h.StartStudentRegistration)
}

func (h *HTTP) StartStudentRegistration(w http.ResponseWriter, r *http.Request) {
	var body PostV1RegistrationStartStudentJSONRequestBody
	if err := httpx.ReadJSON(w, r, &body); err != nil {
		httpx.BadRequest(w, r, err.Error())
		return
	}

	cmd := cmd.StartStudent{Email: string(body.Email)}
	if err := h.cmd.StartStudent.Handle(r.Context(), cmd); err != nil {
		httpx.HandleError(w, r, err)
		return
	}

	httpx.Success(w, r, http.StatusAccepted, nil)
}
