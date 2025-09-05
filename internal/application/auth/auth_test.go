package authapp_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authapp "gitlab.com/ucmsv2/ucms-backend/internal/application/auth"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/builders"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/fixtures"
	"gitlab.com/ucmsv2/ucms-backend/tests/mocks"
)

type AppSuite struct {
	App                     *authapp.App
	MockUserRepo            *mocks.UserRepo
	AccessTokenExpDuration  time.Duration
	RefreshTokenExpDuration time.Duration
	AccessTokenSecretKey    []byte
	RefreshTokenSecretKey   []byte
}

func NewSuite(t *testing.T) *AppSuite {
	t.Helper()

	MockUserRepo := mocks.NewUserRepo()

	accessTokenExp := 15 * time.Minute
	refreshTokenExp := 30 * 24 * time.Hour // 30 days

	return &AppSuite{
		App: authapp.NewApp(authapp.Args{
			UserGetter:              MockUserRepo,
			AccessTokenSecretKey:    fixtures.AccessTokenSecretKey,
			RefreshTokenSecretKey:   fixtures.RefreshTokenSecretKey,
			AccessTokenlExpDuration: &accessTokenExp,
			RefreshTokenExpDuration: &refreshTokenExp,
		}),
		MockUserRepo:            MockUserRepo,
		AccessTokenExpDuration:  accessTokenExp,
		RefreshTokenExpDuration: refreshTokenExp,
		AccessTokenSecretKey:    []byte(fixtures.AccessTokenSecretKey),
		RefreshTokenSecretKey:   []byte(fixtures.RefreshTokenSecretKey),
	}
}

func (a *AppSuite) assertAccessToken(t *testing.T, token, uid, role string) {
	t.Helper()
	authapp.NewJWTTokenAssertion(t, token, a.AccessTokenSecretKey).
		AssertValid().
		AssertISS(authapp.ISS).
		AssertSub(authapp.UserSubject).
		AssertExp(time.Now().Add(a.AccessTokenExpDuration)).
		AssertIAT(time.Now()).
		AssertUID(uid).
		AssertUserRole(role)
}

func (a *AppSuite) assertRefreshToken(t *testing.T, token, uid string) {
	t.Helper()
	authapp.NewJWTTokenAssertion(t, token, a.RefreshTokenSecretKey).
		AssertValid().
		AssertISS(authapp.ISS).
		AssertSub(authapp.RefreshSubject).
		AssertExp(time.Now().Add(a.RefreshTokenExpDuration)).
		AssertIAT(time.Now()).
		AssertUID(uid).
		AssertJTINotEmpty().
		AssertScope(authapp.RefreshScope)
}

func TestLoginHandle_HappyPath(t *testing.T) {
	t.Parallel()

	s := NewSuite(t)
	password := fixtures.TestStudent.Password
	u := builders.NewUserBuilder().WithPassword(password).Build()
	s.MockUserRepo.SeedUser(t, u)

	t.Run("with email", func(t *testing.T) {
		res, err := s.App.LoginHandle(t.Context(), authapp.Login{
			EmailOrBarcode: u.Email(),
			IsEmail:        true,
			Password:       password,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res.AccessToken)
		s.assertAccessToken(t, res.AccessToken, u.ID().String(), u.Role().String())

		require.NotEmpty(t, res.RefreshToken)
		s.assertRefreshToken(t, res.RefreshToken, u.ID().String())
	})

	t.Run("with barcode", func(t *testing.T) {
		res, err := s.App.LoginHandle(t.Context(), authapp.Login{
			EmailOrBarcode: u.Barcode().String(),
			IsEmail:        false,
			Password:       password,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res.AccessToken)
		s.assertAccessToken(t, res.AccessToken, u.ID().String(), u.Role().String())

		require.NotEmpty(t, res.RefreshToken)
		s.assertRefreshToken(t, res.RefreshToken, u.ID().String())
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
		cmd            authapp.Login
		expectedErr    error
		errAssertionFn func(t *testing.T, err error)
	}{
		{
			name: "empty email",
			cmd: authapp.Login{
				EmailOrBarcode: "",
				IsEmail:        true,
				Password:       password,
			},
			expectedErr: authapp.ErrWrongEmailOrBarcodeOrPassword,
		},
		{
			name: "empty barcode",
			cmd: authapp.Login{
				EmailOrBarcode: "",
				IsEmail:        false,
				Password:       password,
			},
			expectedErr: authapp.ErrWrongEmailOrBarcodeOrPassword,
		},
		{
			name: "valid email, but IsEmail is false",
			cmd: authapp.Login{
				EmailOrBarcode: u.Email(),
				IsEmail:        false,
				Password:       password,
			},
			expectedErr: authapp.ErrWrongEmailOrBarcodeOrPassword,
		},
		{
			name: "valid barcode, but IsEmail is true",
			cmd: authapp.Login{
				EmailOrBarcode: u.Barcode().String(),
				IsEmail:        true,
				Password:       password,
			},
			expectedErr: authapp.ErrWrongEmailOrBarcodeOrPassword,
		},
		{
			name: "invalid password, but valid email",
			cmd: authapp.Login{
				EmailOrBarcode: u.Email(),
				IsEmail:        true,
				Password:       wrongPassword,
			},
			expectedErr: authapp.ErrWrongEmailOrBarcodeOrPassword,
		},
		{
			name: "invalid password, but valid barcode",
			cmd: authapp.Login{
				EmailOrBarcode: u.Barcode().String(),
				IsEmail:        false,
				Password:       wrongPassword,
			},
			expectedErr: authapp.ErrWrongEmailOrBarcodeOrPassword,
		},
		{
			name: "non-existent user by email",
			cmd: authapp.Login{
				EmailOrBarcode: email2,
				IsEmail:        true,
				Password:       password,
			},
			expectedErr: authapp.ErrWrongEmailOrBarcodeOrPassword,
		},
		{
			name: "non-existent user by barcode",
			cmd: authapp.Login{
				EmailOrBarcode: "non-existent-barcode",
				IsEmail:        false,
				Password:       password,
			},
			expectedErr: authapp.ErrWrongEmailOrBarcodeOrPassword,
		},
		{
			name: "empty password",
			cmd: authapp.Login{
				EmailOrBarcode: u.Email(),
				IsEmail:        true,
				Password:       "",
			},
			expectedErr: authapp.ErrWrongEmailOrBarcodeOrPassword,
		},
		{
			name: "empty password with barcode",
			cmd: authapp.Login{
				EmailOrBarcode: u.Barcode().String(),
				IsEmail:        false,
				Password:       "",
			},
			expectedErr: authapp.ErrWrongEmailOrBarcodeOrPassword,
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

	loginRes, err := s.App.LoginHandle(t.Context(), authapp.Login{
		EmailOrBarcode: u.Email(),
		IsEmail:        true,
		Password:       password,
	})
	require.NoError(t, err)

	t.Run("valid refresh token", func(t *testing.T) {
		res, err := s.App.RefreshHandle(t.Context(), authapp.Refresh{RefreshToken: loginRes.RefreshToken})
		require.NoError(t, err)
		require.NotNil(t, res)

		assert.Equal(t, loginRes.RefreshToken, res.RefreshToken)

		s.assertAccessToken(t, res.AccessToken, u.ID().String(), u.Role().String())
	})
}

func TestRefreshHandle_FailPath(t *testing.T) {
	s := NewSuite(t)
	uid := fixtures.TestStudent.ID
	password := fixtures.TestStudent.Password
	u := builders.NewUserBuilder().WithID(uid).WithPassword(password).Build()
	s.MockUserRepo.SeedUser(t, u)

	assertInvalidCredential := func(t *testing.T, err error) {
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
				RefreshTokenBuilder(uid.String()).
				WithSecret([]byte("wrong-secret")).
				BuildSignedStringT(t),
			errAssertionFn: assertInvalidCredential,
		},
		{
			name: "expired token",
			refreshToken: builders.JWTFactory{}.
				RefreshTokenBuilder(uid.String()).
				WithExpiration(time.Now().Add(-time.Hour)).
				BuildSignedStringT(t),
			errAssertionFn: assertInvalidCredential,
		},
		{
			name: "empty claims",
			refreshToken: builders.JWTFactory{}.
				RefreshTokenBuilder(uid.String()).
				WithEmptyClaims().
				BuildSignedStringT(t),
			errAssertionFn: assertInvalidCredential,
		},
		{
			name: "user not found",
			refreshToken: builders.JWTFactory{}.
				RefreshTokenBuilder(fixtures.TestStudent2.ID.String()).
				BuildSignedStringT(t),
			errAssertionFn: func(t *testing.T, err error) {
				assert.True(t, errorx.IsCode(err, errorx.CodeInternal), "expected internal error for user not found, got: %v", err)
			},
		},
		{
			name: "invalid iss claim",
			refreshToken: builders.JWTFactory{}.
				RefreshTokenBuilder(uid.String()).
				WithClaim("iss", "invalid_issuer").
				BuildSignedStringT(t),
			errAssertionFn: assertInvalidCredential,
		},
		{
			name: "invalid sub claim",
			refreshToken: builders.JWTFactory{}.
				RefreshTokenBuilder(uid.String()).
				WithClaim("sub", "invalid_subject").
				BuildSignedStringT(t),
			errAssertionFn: assertInvalidCredential,
		},
		{
			name: "missing uid claim",
			refreshToken: builders.JWTFactory{}.
				RefreshTokenBuilder(uid.String()).
				WithClaimEmpty("uid").
				BuildSignedStringT(t),
			errAssertionFn: assertInvalidCredential,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res, err := s.App.RefreshHandle(t.Context(), authapp.Refresh{RefreshToken: tt.refreshToken})
			require.Error(t, err)
			tt.errAssertionFn(t, err)

			assert.Empty(t, res)
		})
	}
}
