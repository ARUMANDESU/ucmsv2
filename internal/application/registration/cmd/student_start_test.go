package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/pkg/env"
	"github.com/ARUMANDESU/ucms/tests/integration/builders"
	"github.com/ARUMANDESU/ucms/tests/integration/fixtures"
	"github.com/ARUMANDESU/ucms/tests/mocks"
)

type StudentStartTestSuite struct {
	Handler  *StartStudentHandler
	MockRepo *mocks.RegistrationRepo
	MockUser *mocks.UserRepo
}

func NewStudentStartTestSuite(t *testing.T) *StudentStartTestSuite {
	t.Helper()

	mockRepo := mocks.NewRegistrationRepo()
	mockUser := mocks.NewUserRepo()
	handler := NewStartStudentHandler(StartStudentHandlerArgs{
		Mode:       env.Test,
		Repo:       mockRepo,
		UserGetter: mockUser,
	})

	return &StudentStartTestSuite{
		Handler:  handler,
		MockRepo: mockRepo,
		MockUser: mockUser,
	}
}

func TestStartStudentHandler_HappyPath(t *testing.T) {
	t.Parallel()

	s := NewStudentStartTestSuite(t)
	email := fixtures.ValidStudentEmail

	err := s.Handler.Handle(t.Context(), StartStudent{Email: email})
	require.NoError(t, err)

	s.MockRepo.
		AssertRegistrationExistsByEmail(t, email).
		AssertStatus(t, registration.StatusPending).
		AssertEmail(t, email).
		AssertVerificationCodeNotEmpty(t)

	s.MockRepo.AssertEventCount(t, 1)
	e := mocks.RequireEventExists(t, s.MockRepo.EventRepo, &registration.RegistrationStarted{})
	require.NotNil(t, e)

	reg, err := s.MockRepo.GetRegistrationByEmail(t.Context(), email)
	require.NoError(t, err)

	assert.Equal(t, reg.ID(), e.RegistrationID)
	assert.Equal(t, email, e.Email)
	assert.Equal(t, reg.VerificationCode(), e.VerificationCode)
}

func TestStartStudentHandler_UserAlreadyExists_MustReturnError(t *testing.T) {
	t.Parallel()
	s := NewStudentStartTestSuite(t)
	u := builders.NewUserBuilder().AsStudent().Build()
	s.MockUser.SeedUser(t, u)

	err := s.Handler.Handle(t.Context(), StartStudent{Email: u.Email()})
	require.Error(t, err)
	// assert.ErrorIs(t, err, apperr.ErrConflict)

	s.MockRepo.AssertRegistrationNotExistsByEmail(t, u.Email())
}

func TestStartStudentHandler_RegistrationCompleted_MustReturnError(t *testing.T) {
	t.Parallel()

	s := NewStudentStartTestSuite(t)
	email := fixtures.ValidStudentEmail
	reg := builders.NewRegistrationBuilder().
		WithEmail(email).
		WithStatus(registration.StatusCompleted).
		Build()
	s.MockRepo.SeedRegistration(t, reg)

	err := s.Handler.Handle(t.Context(), StartStudent{Email: email})
	require.Error(t, err)
	// assert.ErrorIs(t, err, apperr.ErrConflict)

	s.MockRepo.AssertRegistrationExistsByEmail(t, email).
		AssertStatus(t, registration.StatusCompleted).
		AssertEmail(t, email).
		AssertVerificationCodeNotEmpty(t)
}

func TestStartStudentHandler_RegistrationAlreadyExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status registration.Status
	}{
		{
			name:   "Pending",
			status: registration.StatusPending,
		},
		{
			name:   "Expired",
			status: registration.StatusExpired,
		},
		{
			name:   "Verified",
			status: registration.StatusVerified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("resend timeout is not expired, should return error", func(t *testing.T) {
				s := NewStudentStartTestSuite(t)
				email := fixtures.ValidStudentEmail
				reg := builders.NewRegistrationBuilder().
					WithEmail(email).
					WithStatus(tt.status).
					Build()
				s.MockRepo.SeedRegistration(t, reg)

				err := s.Handler.Handle(t.Context(), StartStudent{Email: email})
				require.Error(t, err)
			})

			t.Run("resend timeout is expired, should resend verification code", func(t *testing.T) {
				s := NewStudentStartTestSuite(t)
				email := fixtures.ValidStudentEmail
				reg := builders.NewRegistrationBuilder().
					WithEmail(email).
					WithStatus(tt.status).
					WithResendAvailable().
					Build()
				s.MockRepo.SeedRegistration(t, reg)

				err := s.Handler.Handle(t.Context(), StartStudent{Email: email})
				require.NoError(t, err)

				s.MockRepo.
					AssertRegistrationExistsByEmail(t, email).
					AssertStatus(t, registration.StatusPending).
					AssertEmail(t, email).
					AssertVerificationCodeNotEmpty(t)
				s.MockRepo.AssertEventCount(t, 1)

				e := mocks.RequireEventExists(t, s.MockRepo.EventRepo, &registration.VerificationCodeResent{})
				require.NotNil(t, e)

				reg, err = s.MockRepo.GetRegistrationByEmail(t.Context(), email)
				require.NoError(t, err)
				assert.Equal(t, reg.ID(), e.RegistrationID)
				assert.Equal(t, email, e.Email)
				assert.Equal(t, reg.VerificationCode(), e.VerificationCode)
			})
		})
	}
}

func TestStartStudentHandler_RegistrationAlreadyExists_StatusExpired(t *testing.T) {
	t.Parallel()

	s := NewStudentStartTestSuite(t)
	email := fixtures.ValidStudentEmail
	reg := builders.NewRegistrationBuilder().
		WithEmail(email).
		WithStatus(registration.StatusExpired).
		Build()
	s.MockRepo.SeedRegistration(t, reg)

	err := s.Handler.Handle(t.Context(), StartStudent{Email: email})
	require.Error(t, err)
	// assert.ErrorIs(t, err, apperr.ErrConflict)

	s.MockRepo.AssertRegistrationExistsByEmail(t, email).
		AssertStatus(t, registration.StatusExpired).
		AssertEmail(t, email).
		AssertVerificationCodeNotEmpty(t)
}
