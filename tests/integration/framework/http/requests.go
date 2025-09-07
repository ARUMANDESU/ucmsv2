package http

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	authhttp "gitlab.com/ucmsv2/ucms-backend/internal/ports/http/auth"
	registrationhttp "gitlab.com/ucmsv2/ucms-backend/internal/ports/http/registration"
	staffhttp "gitlab.com/ucmsv2/ucms-backend/internal/ports/http/staff"
)

var ApplicationJSONHeaders = map[string]string{"Content-Type": "application/json"}

func (h *Helper) StartStudentRegistration(t *testing.T, email string) *Response {
	return h.Do(t, NewRequest("POST", "/v1/registrations/students/start").
		WithJSON(map[string]string{"email": email}).
		Build(),
	)
}

func (h *Helper) VerifyRegistrationCode(t *testing.T, email, code string) *Response {
	return h.Do(t, NewRequest("POST", "/v1/registrations/verify").
		WithJSON(registrationhttp.VerifyRequest{
			Email:            email,
			VerificationCode: code,
		}).
		Build(),
	)
}

func (h *Helper) CompleteStudentRegistration(t *testing.T, req registrationhttp.CompleteStudentRegistrationRequest) *Response {
	return h.Do(t, NewRequest("POST", "/v1/registrations/students/complete").
		WithJSON(req).
		Build(),
	)
}

func (h *Helper) ResendVerificationCode(t *testing.T, email string) *Response {
	return h.Do(t, NewRequest("POST", "/v1/registrations/resend").
		WithJSON(map[string]string{"email": email}).
		Build(),
	)
}

func (h *Helper) Login(t *testing.T, emailOrBarcode, password string) *Response {
	return h.Do(t, NewRequest("POST", "/v1/auth/login").
		WithJSON(map[string]string{
			"email_barcode": emailOrBarcode,
			"password":      password,
		}).
		Build(),
	)
}

func (h *Helper) Refresh(t *testing.T, refreshToken string) *Response {
	return h.Do(t, NewRequest("POST", "/v1/auth/refresh").
		WithCookies([]string{
			(&http.Cookie{
				Name:     authhttp.RefreshJWTCookie,
				Value:    refreshToken,
				Path:     authhttp.RefreshCookiePath,
				Domain:   "localhost",
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			}).String(),
		}).
		Build())
}

func (h *Helper) GetVerificationCode(t *testing.T, email string) *Response {
	return h.Do(t, NewRequest("GET", "/dev/registrations/verification-code/"+email).
		Build(),
	)
}

func (h *Helper) Logout(t *testing.T, accessToken, refreshToken string) *Response {
	return h.Do(t, NewRequest("POST", "/v1/auth/logout").
		WithCookies([]string{
			(&http.Cookie{
				Name:     authhttp.RefreshJWTCookie,
				Value:    refreshToken,
				Path:     authhttp.RefreshCookiePath,
				Domain:   "localhost",
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			}).String(),
			(&http.Cookie{
				Name:     authhttp.AccessJWTCookie,
				Value:    accessToken,
				Path:     "/",
				Domain:   "localhost",
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			}).String(),
		}).Build(),
	)
}

func (h *Helper) CreateStaffInvitation(t *testing.T, req staffhttp.CreateInvitationRequest, opts ...RequestBuilderOptions) *Response {
	t.Helper()
	r := NewRequest("POST", "/v1/staffs/invitations").WithJSON(req)
	for _, opt := range opts {
		opt(r)
	}
	return h.Do(t, r.Build())
}

func (h *Helper) UpdateStaffInvitationRecipients(
	t *testing.T,
	invitationID string,
	req staffhttp.UpdateInvitationRecipientsRequest,
	opts ...RequestBuilderOptions,
) *Response {
	t.Helper()
	r := NewRequest("PUT", "/v1/staffs/invitations/"+invitationID+"/recipients").WithJSON(req)
	for _, opt := range opts {
		opt(r)
	}
	return h.Do(t, r.Build())
}

func (h *Helper) UpdateStaffInvitationValidity(
	t *testing.T,
	invitationID string,
	req staffhttp.UpdateInvitationValidityRequest,
	opts ...RequestBuilderOptions,
) *Response {
	t.Helper()
	r := NewRequest("PUT", "/v1/staffs/invitations/"+invitationID+"/validity").WithJSON(req)
	for _, opt := range opts {
		opt(r)
	}
	return h.Do(t, r.Build())
}

func (h *Helper) DeleteStaffInvitation(t *testing.T, invitationID string, opts ...RequestBuilderOptions) *Response {
	t.Helper()
	r := NewRequest("DELETE", "/v1/staffs/invitations/"+invitationID)
	for _, opt := range opts {
		opt(r)
	}
	return h.Do(t, r.Build())
}

func (h *Helper) ValidateStaffInvitation(t *testing.T, code string, email string, opts ...RequestBuilderOptions) *Response {
	t.Helper()
	r := NewRequest("GET", fmt.Sprintf("/v1/invitations/%s/validate?email=%s", code, email))
	for _, opt := range opts {
		opt(r)
	}
	return h.Do(t, r.Build())
}

func (h *Helper) AcceptStaffInvitation(t *testing.T, req staffhttp.AcceptInvitationRequest, opts ...RequestBuilderOptions) *Response {
	t.Helper()
	r := NewRequest("POST", "/v1/invitations/accept").WithJSON(req)
	for _, opt := range opts {
		opt(r)
	}
	return h.Do(t, r.Build())
}

func (h *Helper) UpdateUserAvatar(t *testing.T, fileData []byte, opts ...RequestBuilderOptions) *Response {
	var body io.Reader
	var contentType string

	if fileData != nil {
		body, contentType = NewMultipartFormBuilder().AddFile("avatar", "avatar.jpg", "image/jpeg", fileData).Build()
	}

	req := NewRequest("PATCH", "/v1/users/me/avatar")
	if body != nil {
		req.WithBody(body).WithHeader("Content-Type", contentType)
	}

	for _, opt := range opts {
		opt(req)
	}

	return h.Do(t, req.Build())
}

func (h *Helper) UpdateUserAvatarWithFile(t *testing.T, filename, contentType string, fileData []byte, opts ...RequestBuilderOptions) *Response {
	body, formContentType := NewMultipartFormBuilder().AddFile("avatar", filename, contentType, fileData).Build()

	req := NewRequest("PATCH", "/v1/users/me/avatar").
		WithBody(body).
		WithHeader("Content-Type", formContentType)

	for _, opt := range opts {
		opt(req)
	}

	return h.Do(t, req.Build())
}
