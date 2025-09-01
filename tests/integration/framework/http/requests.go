package http

import (
	"net/http"
	"testing"

	authhttp "github.com/ARUMANDESU/ucms/internal/ports/http/auth"
	registrationhttp "github.com/ARUMANDESU/ucms/internal/ports/http/registration"
	staffhttp "github.com/ARUMANDESU/ucms/internal/ports/http/staff"
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

func (h *Helper) CreateStaffInvitationRequest(req staffhttp.CreateInvitationRequest) *RequestBuilder {
	return NewRequest("POST", "/v1/staffs/invitations").WithJSON(req)
}

func (h *Helper) UpdateStaffInvitationRecipientsRequest(invitationID string, req staffhttp.UpdateInvitationRecipientsRequest) *RequestBuilder {
	return NewRequest("PUT", "/v1/staffs/invitations/"+invitationID+"/recipients").WithJSON(req)
}
