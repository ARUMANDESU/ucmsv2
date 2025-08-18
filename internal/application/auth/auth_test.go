package authapp

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/tests/integration/builders"
	"github.com/ARUMANDESU/ucms/tests/integration/fixtures"
	"github.com/ARUMANDESU/ucms/tests/mocks"
)

type AppSuite struct {
	App          *App
	MockUserRepo *mocks.UserRepo
}

func NewSuite(t *testing.T) *AppSuite {
	t.Helper()

	MockUserRepo := mocks.NewUserRepo()

	accessTokenExp := 15 * time.Minute
	refreshTokenExp := 30 * 24 * time.Hour // 30 days

	return &AppSuite{
		App: NewApp(Args{
			UserGetter:              MockUserRepo,
			AccessTokenSecretKey:    "secret1",
			RefreshTokenSecretKey:   "secret2",
			AccessTokenlExpDuration: &accessTokenExp,
			RefreshTokenExpDuration: &refreshTokenExp,
		}),
		MockUserRepo: MockUserRepo,
	}
}

func (a *AppSuite) assertAccessToken(t *testing.T, token, uid, role string) {
	t.Helper()
	NewJWTTokenAssertion(t, token, a.App.accessTokenSecretKey).
		AssertValid().
		AssertISS("ucmsv2_auth").
		AssertSub("user").
		AssertExp(time.Now().Add(a.App.accessTokenExpDuration)).
		AssertIAT(time.Now()).
		AssertUID(uid).
		AssertUserRole(role)
}

func (a *AppSuite) assertRefreshToken(t *testing.T, token, uid string) {
	t.Helper()
	NewJWTTokenAssertion(t, token, a.App.refreshTokenSecretKey).
		AssertValid().
		AssertISS("ucmsv2_auth").
		AssertSub("refresh").
		AssertExp(time.Now().Add(a.App.refreshTokenExpDuration)).
		AssertIAT(time.Now()).
		AssertUID(uid).
		AssertJTINotEmpty().
		AssertScope("refresh")
}

func TestLoginHandle_HappyPath(t *testing.T) {
	t.Parallel()

	s := NewSuite(t)
	password := fixtures.TestStudent.Password
	u := builders.NewUserBuilder().WithPassword(password).Build()
	s.MockUserRepo.SeedUser(t, u)

	t.Run("with email", func(t *testing.T) {
		res, err := s.App.LoginHandle(t.Context(), Login{
			EmailOrBarcode: u.Email(),
			IsEmail:        true,
			Password:       password,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res.AccessToken)
		s.assertAccessToken(t, res.AccessToken, u.Barcode().String(), u.Role().String())

		require.NotEmpty(t, res.RefreshToken)
		s.assertRefreshToken(t, res.RefreshToken, u.Barcode().String())
	})

	t.Run("with barcode", func(t *testing.T) {
		res, err := s.App.LoginHandle(t.Context(), Login{
			EmailOrBarcode: u.Barcode().String(),
			IsEmail:        false,
			Password:       password,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res.AccessToken)
		s.assertAccessToken(t, res.AccessToken, u.Barcode().String(), u.Role().String())

		require.NotEmpty(t, res.RefreshToken)
		s.assertRefreshToken(t, res.RefreshToken, u.Barcode().String())
	})
}

func TestLoginHandle_FailPath(t *testing.T) {
	s := NewSuite(t)
	password := fixtures.TestStudent.Password
	wrongPassword := fixtures.TestStudent2.Password
	email := fixtures.TestStudent.Email
	email2 := fixtures.TestStudent2.Email
	u := builders.NewUserBuilder().
		WithEmail(email).
		WithPassword(password).
		Build()
	s.MockUserRepo.SeedUser(t, u)

	// notFoundAssertion := func(t *testing.T, err error) {
	// 	assert.True(t, errorx.IsNotFound(err), "expected not found error, got: %v", err)
	// }

	tests := []struct {
		name           string
		cmd            Login
		expectedErr    error
		errAssertionFn func(t *testing.T, err error)
	}{
		{
			name: "empty email",
			cmd: Login{
				EmailOrBarcode: "",
				IsEmail:        true,
				Password:       password,
			},
			expectedErr: ErrWrongEmailOrBarcodeOrPassword,
		},
		{
			name: "empty barcode",
			cmd: Login{
				EmailOrBarcode: "",
				IsEmail:        false,
				Password:       password,
			},
			expectedErr: ErrWrongEmailOrBarcodeOrPassword,
		},
		{
			name: "valid email, but IsEmail is false",
			cmd: Login{
				EmailOrBarcode: u.Email(),
				IsEmail:        false,
				Password:       password,
			},
			expectedErr: ErrWrongEmailOrBarcodeOrPassword,
		},
		{
			name: "valid barcode, but IsEmail is true",
			cmd: Login{
				EmailOrBarcode: u.Barcode().String(),
				IsEmail:        true,
				Password:       password,
			},
			expectedErr: ErrWrongEmailOrBarcodeOrPassword,
		},
		{
			name: "invalid password, but valid email",
			cmd: Login{
				EmailOrBarcode: u.Email(),
				IsEmail:        true,
				Password:       wrongPassword,
			},
			expectedErr: ErrWrongEmailOrBarcodeOrPassword,
		},
		{
			name: "invalid password, but valid barcode",
			cmd: Login{
				EmailOrBarcode: u.Barcode().String(),
				IsEmail:        false,
				Password:       wrongPassword,
			},
			expectedErr: ErrWrongEmailOrBarcodeOrPassword,
		},
		{
			name: "non-existent user by email",
			cmd: Login{
				EmailOrBarcode: email2,
				IsEmail:        true,
				Password:       password,
			},
			expectedErr: ErrWrongEmailOrBarcodeOrPassword,
		},
		{
			name: "non-existent user by barcode",
			cmd: Login{
				EmailOrBarcode: "non-existent-barcode",
				IsEmail:        false,
				Password:       password,
			},
			expectedErr: ErrWrongEmailOrBarcodeOrPassword,
		},
		{
			name: "empty password",
			cmd: Login{
				EmailOrBarcode: u.Email(),
				IsEmail:        true,
				Password:       "",
			},
			expectedErr: ErrWrongEmailOrBarcodeOrPassword,
		},
		{
			name: "empty password with barcode",
			cmd: Login{
				EmailOrBarcode: u.Barcode().String(),
				IsEmail:        false,
				Password:       "",
			},
			expectedErr: ErrWrongEmailOrBarcodeOrPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := s.App.LoginHandle(t.Context(), tt.cmd)
			require.Error(t, err)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else if tt.errAssertionFn != nil {
				tt.errAssertionFn(t, err)
			} else {
				t.Fatalf("no expected error or assertion function provided for test case: %s", tt.name)
			}

			assert.NotNil(t, res)
		})
	}
}

func TestRefreshHandle_HappyPath(t *testing.T) {
	s := NewSuite(t)
	password := fixtures.TestStudent.Password
	u := builders.NewUserBuilder().WithPassword(password).Build()
	s.MockUserRepo.SeedUser(t, u)

	loginRes, err := s.App.LoginHandle(t.Context(), Login{
		EmailOrBarcode: u.Email(),
		IsEmail:        true,
		Password:       password,
	})
	require.NoError(t, err)

	t.Run("valid refresh token", func(t *testing.T) {
		res, err := s.App.RefreshHandle(t.Context(), Refresh{RefreshToken: loginRes.RefreshToken})
		require.NoError(t, err)
		require.NotNil(t, res)

		assert.Equal(t, loginRes.RefreshToken, res.RefreshToken)

		s.assertAccessToken(t, res.AccessToken, u.Barcode().String(), u.Role().String())
	})
}

func TestRefreshHandle_FailPath(t *testing.T) {
	s := NewSuite(t)
	uid := fixtures.TestStudent.Barcode
	password := fixtures.TestStudent.Password
	u := builders.NewUserBuilder().WithBarcode(uid).WithPassword(password).Build()
	s.MockUserRepo.SeedUser(t, u)

	assertInvalidCredential := func(t *testing.T, err error) {
		fmt.Printf("test case: %s, error: %v\n", t.Name(), err)
		assert.True(t, errorx.IsCode(err, errorx.CodeInvalidCredentials), "expected invalid credentials error, got: %v", err)
	}

	tests := []struct {
		name           string
		refreshToken   string
		errAssertionFn func(t *testing.T, err error)
	}{
		{
			name: "invalid signature",
			refreshToken: builders.JWTFactory{}.
				RefreshTokenBuilder(uid).
				WithSecret([]byte("wrong-secret")).
				BuildSignedStringT(t),
			errAssertionFn: assertInvalidCredential,
		},
		{
			name: "expired token",
			refreshToken: builders.JWTFactory{}.
				RefreshTokenBuilder(uid).
				WithExpiration(time.Now().Add(-time.Hour)).
				BuildSignedStringT(t),
			errAssertionFn: assertInvalidCredential,
		},
		{
			name: "empty claims",
			refreshToken: builders.JWTFactory{}.
				RefreshTokenBuilder(uid).
				WithEmptyClaims().
				BuildSignedStringT(t),
			errAssertionFn: assertInvalidCredential,
		},
		{
			name: "user not found",
			refreshToken: builders.JWTFactory{}.
				RefreshTokenBuilder(fixtures.TestStudent2.Barcode).
				BuildSignedStringT(t),
			errAssertionFn: func(t *testing.T, err error) {
				assert.True(t, errorx.IsCode(err, errorx.CodeInternal), "expected internal error for user not found, got: %v", err)
			},
		},
		{
			name: "invalid iss claim",
			refreshToken: builders.JWTFactory{}.
				RefreshTokenBuilder(uid).
				WithClaim("iss", "invalid_issuer").
				BuildSignedStringT(t),
			errAssertionFn: assertInvalidCredential,
		},
		{
			name: "invalid sub claim",
			refreshToken: builders.JWTFactory{}.
				RefreshTokenBuilder(uid).
				WithClaim("sub", "invalid_subject").
				BuildSignedStringT(t),
			errAssertionFn: assertInvalidCredential,
		},
		{
			name: "missing uid claim",
			refreshToken: builders.JWTFactory{}.
				RefreshTokenBuilder(uid).
				WithClaimEmpty("uid").
				BuildSignedStringT(t),
			errAssertionFn: assertInvalidCredential,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res, err := s.App.RefreshHandle(t.Context(), Refresh{RefreshToken: tt.refreshToken})
			require.Error(t, err)
			tt.errAssertionFn(t, err)

			assert.Empty(t, res)
		})
	}
}

type JWTTokenAssertion struct {
	token    string
	jwttoken *jwt.Token
	claims   jwt.MapClaims
	t        *testing.T
}

func NewJWTTokenAssertion(t *testing.T, token string, secretkey []byte) *JWTTokenAssertion {
	t.Helper()

	jwttoken, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		return secretkey, nil
	})
	require.NoError(t, err)

	claims, ok := jwttoken.Claims.(jwt.MapClaims)
	require.True(t, ok, "jwt token claims must be type jwt.MapClaims")

	return &JWTTokenAssertion{
		t:        t,
		token:    token,
		jwttoken: jwttoken,
		claims:   claims,
	}
}

func (a *JWTTokenAssertion) AssertValid() *JWTTokenAssertion {
	a.t.Helper()
	assert.NotNil(a.t, a.jwttoken, "jwt token should not be nil")
	assert.True(a.t, a.jwttoken.Valid, "jwt token should be valid")
	return a
}

func (a *JWTTokenAssertion) AssertNotValid() *JWTTokenAssertion {
	a.t.Helper()
	assert.NotNil(a.t, a.jwttoken, "jwt token should not be nil")
	assert.False(a.t, a.jwttoken.Valid, "jwt token should not be valid")
	return a
}

func (a *JWTTokenAssertion) AssertISS(expected string) *JWTTokenAssertion {
	a.t.Helper()
	assert.Equal(a.t, a.claims["iss"], expected)
	return a
}

func (a *JWTTokenAssertion) AssertSub(expected string) *JWTTokenAssertion {
	a.t.Helper()
	assert.Equal(a.t, a.claims["sub"], expected)
	return a
}

func (a *JWTTokenAssertion) AssertExp(expected time.Time) *JWTTokenAssertion {
	a.t.Helper()
	exp, ok := a.claims["exp"].(float64)
	require.True(a.t, ok, "exp claim must be of type float64, got %T", a.claims["exp"])
	assert.NotZero(a.t, exp, "exp claim should not be zero")
	expTime := time.Unix(int64(exp), 0)
	assert.WithinDuration(a.t, expected, expTime, time.Second, "exp claim should be within 1 second of expected time")
	return a
}

func (a *JWTTokenAssertion) AssertIAT(expected time.Time) *JWTTokenAssertion {
	a.t.Helper()
	iat, ok := a.claims["iat"].(float64)
	require.True(a.t, ok, "iat claim must be of type float64, got %T", a.claims["iat"])

	assert.NotZero(a.t, iat, "iat claim should not be zero")
	iatTime := time.Unix(int64(iat), 0)

	assert.WithinDuration(a.t, expected, iatTime, time.Second, "iat claim should be within 1 second of expected time")
	return a
}

func (a *JWTTokenAssertion) AssertScope(expected string) *JWTTokenAssertion {
	a.t.Helper()
	assert.Equal(a.t, a.claims["scope"], expected)
	return a
}

func (a *JWTTokenAssertion) AssertJTI(expected string) *JWTTokenAssertion {
	a.t.Helper()
	assert.Equal(a.t, a.claims["jti"], expected)
	return a
}

func (a *JWTTokenAssertion) AssertJTINotEmpty() *JWTTokenAssertion {
	a.t.Helper()
	assert.NotEmpty(a.t, a.claims["jti"], "jti claim should not be empty")
	return a
}

func (a *JWTTokenAssertion) AssertUID(expected string) *JWTTokenAssertion {
	a.t.Helper()
	assert.Equal(a.t, a.claims["uid"], expected)
	return a
}

func (a *JWTTokenAssertion) AssertUserRole(expected string) *JWTTokenAssertion {
	a.t.Helper()
	assert.Equal(a.t, a.claims["user_role"], expected)
	return a
}
