package http

import (
	"net/http"
	"testing"

	authhttp "github.com/ARUMANDESU/ucms/internal/ports/http/auth"
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
			Email:            email,
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

func (h *Helper) Login(t *testing.T, emailOrBarcode, password string) *Response {
	return h.Do(t, Request{
		Method: "POST",
		Path:   "/v1/auth/login",
		Body: map[string]string{
			"email_barcode": emailOrBarcode,
			"password":      password,
		},
	})
}

func (h *Helper) Refresh(t *testing.T, refreshToken string) *Response {
	return h.Do(t, NewRequest("POST", "/v1/auth/refresh").
		WithCookies(map[string]string{
			authhttp.RefreshJWTCookie: (&http.Cookie{
				Name:     "ucmsv2_refresh",
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

func (h *Helper) Logout(t *testing.T, accessToken, refreshToken string) *Response {
	return h.Do(t, NewRequest("POST", "/v1/auth/logout").
		WithCookies(map[string]string{
			authhttp.RefreshJWTCookie: (&http.Cookie{
				Name:     "ucmsv2_refresh",
				Value:    refreshToken,
				Path:     authhttp.RefreshCookiePath,
				Domain:   "localhost",
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			}).String(),
			authhttp.AccessJWTCookie: (&http.Cookie{
				Name:     "ucmsv2_access",
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
