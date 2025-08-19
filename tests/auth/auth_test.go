package auth

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	authapp "github.com/ARUMANDESU/ucms/internal/application/auth"
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
	email := fixtures.TestStudent.Email
	barcode := fixtures.TestStudent.Barcode
	password := fixtures.TestStudent.Password
	u := builders.NewUserBuilder().
		WithEmail(email).
		WithBarcode(barcode).
		WithPassword(password).
		Build()
	s.DB.SeedUser(s.T(), u)

	testCases := []struct {
		name         string
		loginField   string
		expectedUID  string
		expectedRole string
	}{
		{
			name:         "login with email",
			loginField:   u.Email(),
			expectedUID:  u.Barcode().String(),
			expectedRole: u.Role().String(),
		},
		{
			name:         "login with barcode",
			loginField:   u.Barcode().String(),
			expectedUID:  u.Barcode().String(),
			expectedRole: u.Role().String(),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			resp := s.HTTP.Login(t, tc.loginField, fixtures.TestStudent.Password)

			resp.AssertSuccess()

			s.assertValidAccessToken(t, resp, tc.expectedUID, tc.expectedRole)
			s.assertValidRefreshToken(t, resp, tc.expectedUID)
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
