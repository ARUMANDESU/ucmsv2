package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/pkg/apperr"
	"github.com/ARUMANDESU/ucms/tests/integration/builders"
	"github.com/ARUMANDESU/ucms/tests/integration/fixtures"
	"github.com/ARUMANDESU/ucms/tests/mocks"
)

type VerifySuite struct {
	Handler  *VerifyHandler
	MockRepo *mocks.RegistrationRepo
}

func NewVerifySuite() *VerifySuite {
	mockRepo := mocks.NewRegistrationRepo()
	handler := NewVerifyHandler(VerifyHandlerArgs{
		RegistrationRepo: mockRepo,
	})

	return &VerifySuite{
		Handler:  handler,
		MockRepo: mockRepo,
	}
}

func TestVerifyHandler_HappyPath(t *testing.T) {
	t.Parallel()

	s := NewVerifySuite()
	reg := builders.NewRegistrationBuilder().Build()
	s.MockRepo.SeedRegistration(t, reg)

	err := s.Handler.Handle(t.Context(), Verify{
		Email: reg.Email(),
		Code:  reg.VerificationCode(),
	})
	require.NoError(t, err)

	s.MockRepo.
		AssertRegistrationExistsByEmail(t, reg.Email()).
		AssertStatus(t, registration.StatusVerified)

	s.MockRepo.AssertEventCount(t, 1)
	e := mocks.RequireEventExists(t, s.MockRepo, &registration.EmailVerified{})
	require.NotNil(t, e)
	assert.Equal(t, reg.ID(), e.RegistrationID)
	assert.Equal(t, reg.Email(), e.Email)
}

func TestVerifyHandler_AlreadyVerified_ShouldSucceed(t *testing.T) {
	t.Parallel()

	s := NewVerifySuite()
	reg := builders.NewRegistrationBuilder().
		WithEmail(fixtures.ValidStudentEmail).
		WithStatus(registration.StatusVerified).
		Build()
	s.MockRepo.SeedRegistration(t, reg)

	err := s.Handler.Handle(t.Context(), Verify{
		Email: reg.Email(),
		Code:  reg.VerificationCode(),
	})
	require.ErrorIs(t, err, ErrOKAlreadyVerified)

	s.MockRepo.AssertEventCount(t, 0)
}

func TestVerifyHandler_InvalidArgs_ShouldReturnError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		arg     Verify
		wantErr error
	}{
		{
			name:    "Empty Email",
			arg:     Verify{Email: "", Code: "valid-code"},
			wantErr: apperr.ErrInvalidInput,
		},
		{
			name:    "Empty Code",
			arg:     Verify{Email: fixtures.ValidStudentEmail, Code: ""},
			wantErr: apperr.ErrInvalidInput,
		},
		{
			name:    "Both Empty",
			arg:     Verify{Email: "", Code: ""},
			wantErr: apperr.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := NewVerifySuite()
			err := s.Handler.Handle(t.Context(), tt.arg)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)

			s.MockRepo.AssertEventCount(t, 0)
		})
	}
}

func TestVerifyHandler_InvalidEmail_ShouldReturnError(t *testing.T) {
	t.Parallel()

	s := NewVerifySuite()
	email1 := fixtures.ValidStudentEmail
	email2 := fixtures.ValidExternalEmail
	reg := builders.NewRegistrationBuilder().WithEmail(email1).Build()
	s.MockRepo.SeedRegistration(t, reg)

	err := s.Handler.Handle(t.Context(), Verify{
		Email: email2,
		Code:  reg.VerificationCode(),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestVerifyHandler_InvalidCode_ShouldReturnError(t *testing.T) {
	t.Parallel()

	s := NewVerifySuite()
	reg := builders.NewRegistrationBuilder().Build()
	s.MockRepo.SeedRegistration(t, reg)

	err := s.Handler.Handle(t.Context(), Verify{
		Email: reg.Email(),
		Code:  "invalid-code",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, registration.ErrInvalidVerificationCode)

	s.MockRepo.AssertRegistrationExistsByEmail(t, reg.Email()).
		AssertStatus(t, registration.StatusPending).
		AssertCodeAttempts(t, 1).
		AssertVerificationCodeNotEmpty(t)
	s.MockRepo.AssertEventCount(t, 0)
}

func TestVerifyHandler_InvalidCode_TooManyAttempts_ShouldReturnError(t *testing.T) {
	t.Parallel()

	s := NewVerifySuite()
	reg := builders.NewRegistrationBuilder().
		WithEmail(fixtures.ValidStudentEmail).
		WithVerificationCode("valid-code").
		WithCodeAttempts(registration.MaxVerificationCodeAttempts).
		Build()
	s.MockRepo.SeedRegistration(t, reg)

	err := s.Handler.Handle(t.Context(), Verify{
		Email: reg.Email(),
		Code:  "invalid-code",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, registration.ErrTooManyAttempts)

	s.MockRepo.AssertRegistrationExistsByEmail(t, reg.Email()).
		AssertStatus(t, registration.StatusExpired).
		AssertCodeAttempts(t, registration.MaxVerificationCodeAttempts+1).
		AssertVerificationCodeNotEmpty(t)
	s.MockRepo.AssertEventCount(t, 0)
}
