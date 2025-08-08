package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/pkg/env"
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
	e := mocks.RequireEventExists(t, s.MockRepo, &registration.RegistrationStarted{})

	require.NotNil(t, e)

	reg, err := s.MockRepo.GetRegistrationByEmail(t.Context(), email)
	require.NoError(t, err)

	assert.Equal(t, reg.ID(), e.RegistrationID)
	assert.Equal(t, email, e.Email)
	assert.Equal(t, reg.VerificationCode(), e.VerificationCode)
}
