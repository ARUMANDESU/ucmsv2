package http

import (
	"testing"

	openapi_types "github.com/oapi-codegen/runtime/types"

	registrationhttp "github.com/ARUMANDESU/ucms/internal/ports/http/registration"
)

func (h *Helper) StartStudentRegistration(t *testing.T, email string) *Response {
	return h.Do(t, Request{
		Method: "POST",
		Path:   "/v1/registrations/students/start",
		Body:   map[string]string{"email": email},
	})
}

func (h *Helper) VerifyRegistrationCode(t *testing.T, email, code string) *Response {
	return h.Do(t, Request{
		Method: "POST",
		Path:   "/v1/registrations/verify",
		Body: registrationhttp.PostV1RegistrationsVerifyJSONRequestBody{
			Email:            openapi_types.Email(email),
			VerificationCode: code,
		},
	})
}

func (h *Helper) CompleteStudentRegistration(t *testing.T, req registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) *Response {
	return h.Do(t, Request{
		Method: "POST",
		Path:   "/v1/registrations/students/complete",
		Body:   req,
	})
}

func (h *Helper) ResendVerificationCode(t *testing.T, email string) *Response {
	return h.Do(t, Request{
		Method: "POST",
		Path:   "/v1/registrations/resend",
		Body:   map[string]string{"email": email},
	})
}
