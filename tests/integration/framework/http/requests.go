package http

import "testing"

func (h *Helper) StartStudentRegistration(t *testing.T, email string) *Response {
	return h.Do(t, Request{
		Method: "POST",
		Path:   "/v1/registration/start/student",
		Body:   map[string]string{"email": email},
	})
}

func (h *Helper) VerifyRegistrationCode(t *testing.T, email, code string) *Response {
	return h.Do(t, Request{
		Method: "POST",
		Path:   "/v1/registration/verify",
		Body: map[string]string{
			"email": email,
			"code":  code,
		},
	})
}

func (h *Helper) CompleteStudentRegistration(t *testing.T, req CompleteRegistrationRequest) *Response {
	return h.Do(t, Request{
		Method: "POST",
		Path:   "/v1/registration/complete/student",
		Body:   req,
	})
}

type CompleteRegistrationRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Password  string `json:"password"`
	GroupID   string `json:"group_id,omitempty"`
}
