package cmd

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/tests/integration/builders"
	"github.com/ARUMANDESU/ucms/tests/integration/fixtures"
	"github.com/ARUMANDESU/ucms/tests/mocks"
)

type StudentCompleteSuite struct {
	Handler          *StudentCompleteHandler
	MockUser         *mocks.UserRepo
	MockRegistration *mocks.RegistrationRepo
	MockGroup        *mocks.GroupRepo
}

func NewStudentCompleteSuite(t *testing.T) *StudentCompleteSuite {
	mockUser := mocks.NewUserRepo()
	mockRegistration := mocks.NewRegistrationRepo()
	mockGroup := mocks.NewGroupRepo()

	// Seed a group for the test
	group := builders.NewGroupBuilder().Build()
	mockGroup.SeedGroup(t.Context(), group)

	handler := NewStudentCompleteHandler(StudentCompleteHandlerArgs{
		UserGetter:       mockUser,
		RegistrationRepo: mockRegistration,
		GroupGetter:      mockGroup,
	})

	return &StudentCompleteSuite{
		Handler:          handler,
		MockUser:         mockUser,
		MockRegistration: mockRegistration,
		MockGroup:        mockGroup,
	}
}

func TestStudentCompleteHandler_HappyPath(t *testing.T) {
	t.Parallel()

	t.Run("already verified registration", func(t *testing.T) {
		s := NewStudentCompleteSuite(t)
		reg := builders.NewRegistrationBuilder().
			WithEmail(fixtures.ValidStudentEmail).
			WithStatus(registration.StatusVerified).
			Build()
		s.MockRegistration.SeedRegistration(t, reg)

		err := s.Handler.Handle(t.Context(), StudentComplete{
			Email:            fixtures.TestStudent.Email,
			VerificationCode: reg.VerificationCode(),
			Barcode:          fixtures.TestStudent.ID,
			FirstName:        fixtures.TestStudent.FirstName,
			LastName:         fixtures.TestStudent.LastName,
			Password:         fixtures.TestStudent.Password,
			GroupID:          fixtures.TestStudent.GroupID,
		})
		require.NoError(t, err)

		s.MockRegistration.
			AssertRegistrationExistsByID(t, reg.ID()).
			AssertStatus(t, registration.StatusCompleted).
			AssertEmail(t, fixtures.ValidStudentEmail).
			AssertVerificationCode(t, reg.VerificationCode())

		s.MockRegistration.AssertEventCount(t, 1)
		e := mocks.RequireEventExists(t, s.MockRegistration.EventRepo, &registration.StudentRegistrationCompleted{})
		assert.Equal(t, reg.ID(), e.RegistrationID)
		assert.Equal(t, fixtures.TestStudent.ID, e.Barcode)
		assert.Equal(t, fixtures.TestStudent.Email, e.Email)
		assert.Equal(t, fixtures.TestStudent.FirstName, e.FirstName)
		assert.Equal(t, fixtures.TestStudent.LastName, e.LastName)
		assert.NoError(t, bcrypt.CompareHashAndPassword(e.PassHash, []byte(fixtures.TestStudent.Password)), "password should match")
		assert.Equal(t, fixtures.TestStudent.GroupID, e.GroupID)
	})

	t.Run("not verified yet, complete registration", func(t *testing.T) {
		s := NewStudentCompleteSuite(t)
		reg := builders.NewRegistrationBuilder().
			WithEmail(fixtures.ValidStudentEmail).
			WithStatus(registration.StatusPending).
			Build()
		s.MockRegistration.SeedRegistration(t, reg)

		err := s.Handler.Handle(t.Context(), StudentComplete{
			Email:            fixtures.TestStudent.Email,
			VerificationCode: reg.VerificationCode(),
			Barcode:          fixtures.TestStudent.ID,
			FirstName:        fixtures.TestStudent.FirstName,
			LastName:         fixtures.TestStudent.LastName,
			Password:         fixtures.TestStudent.Password,
			GroupID:          fixtures.TestStudent.GroupID,
		})
		require.NoError(t, err)
		s.MockRegistration.
			AssertRegistrationExistsByID(t, reg.ID()).
			AssertStatus(t, registration.StatusCompleted).
			AssertEmail(t, fixtures.ValidStudentEmail).
			AssertVerificationCode(t, reg.VerificationCode())

		s.MockRegistration.AssertEventCount(t, 2)

		eventVerified := mocks.RequireEventExists(t, s.MockRegistration.EventRepo, &registration.EmailVerified{})
		assert.Equal(t, reg.ID(), eventVerified.RegistrationID)
		assert.Equal(t, fixtures.TestStudent.Email, eventVerified.Email)

		eventCompleted := mocks.RequireEventExists(t, s.MockRegistration.EventRepo, &registration.StudentRegistrationCompleted{})
		assert.Equal(t, reg.ID(), eventCompleted.RegistrationID)
		assert.Equal(t, fixtures.TestStudent.ID, eventCompleted.Barcode)
		assert.Equal(t, fixtures.TestStudent.Email, eventCompleted.Email)
		assert.Equal(t, fixtures.TestStudent.FirstName, eventCompleted.FirstName)
		assert.Equal(t, fixtures.TestStudent.LastName, eventCompleted.LastName)
		assert.NoError(t, bcrypt.CompareHashAndPassword(eventCompleted.PassHash, []byte(fixtures.TestStudent.Password)), "password should match")
		assert.Equal(t, fixtures.TestStudent.GroupID, eventCompleted.GroupID)
	})
}

func TestStudentCompleteHandler_UserAlreadyExists_ShouldFail(t *testing.T) {
	t.Parallel()

	t.Run("user by email already exists", func(t *testing.T) {
		s := NewStudentCompleteSuite(t)
		u := builders.NewStudentBuilder().Build().User()
		s.MockUser.SeedUser(t, u)

		err := s.Handler.Handle(t.Context(), StudentComplete{
			Email:            u.Email(),
			VerificationCode: fixtures.ValidVerificationCode,
			Barcode:          string(u.ID()),
			FirstName:        u.FirstName(),
			LastName:         u.LastName(),
			Password:         fixtures.TestStudent.Password,
			GroupID:          fixtures.TestStudent.GroupID,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmailNotAvailable)
	})

	t.Run("user by barcode already exists", func(t *testing.T) {
		s := NewStudentCompleteSuite(t)
		u := builders.NewStudentBuilder().WithEmail(fixtures.TestStudent2.Email).Build().User()
		s.MockUser.SeedUser(t, u)

		err := s.Handler.Handle(t.Context(), StudentComplete{
			Email:            fixtures.TestStudent.Email,
			VerificationCode: fixtures.ValidVerificationCode,
			Barcode:          string(u.ID()),
			FirstName:        fixtures.TestStudent.FirstName,
			LastName:         fixtures.TestStudent.LastName,
			Password:         fixtures.TestStudent.Password,
			GroupID:          fixtures.TestStudent.GroupID,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrBarcodeNotAvailable)
	})
	t.Run("user by email and barcode already exists", func(t *testing.T) {
		s := NewStudentCompleteSuite(t)
		u := builders.NewStudentBuilder().Build().User()
		s.MockUser.SeedUser(t, u)

		err := s.Handler.Handle(t.Context(), StudentComplete{
			Email:            u.Email(),
			VerificationCode: fixtures.ValidVerificationCode,
			Barcode:          string(u.ID()),
			FirstName:        fixtures.TestStudent.FirstName,
			LastName:         fixtures.TestStudent.LastName,
			Password:         fixtures.TestStudent.Password,
			GroupID:          fixtures.TestStudent.GroupID,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmailNotAvailable)
	})

	t.Run("group not found", func(t *testing.T) {
		s := NewStudentCompleteSuite(t)
		reg := builders.NewRegistrationBuilder().
			WithEmail(fixtures.ValidStudentEmail).
			WithStatus(registration.StatusPending).
			Build()
		s.MockRegistration.SeedRegistration(t, reg)
		err := s.Handler.Handle(t.Context(), StudentComplete{
			Email:            fixtures.TestStudent.Email,
			VerificationCode: reg.VerificationCode(),
			Barcode:          fixtures.TestStudent.ID,
			FirstName:        fixtures.TestStudent.FirstName,
			LastName:         fixtures.TestStudent.LastName,
			Password:         fixtures.TestStudent.Password,
			GroupID:          uuid.New(), // Use a non-existing group ID
		})
		require.Error(t, err)
		assert.True(t, errorx.IsNotFound(err), "expected not found error, got: %v", err)
	})
}

func TestStudentCompleteHandler_Verified(t *testing.T) {
	t.Parallel()

	t.Run("with invalid verification code", func(t *testing.T) {
		s := NewStudentCompleteSuite(t)
		reg := builders.NewRegistrationBuilder().
			WithEmail(fixtures.TestStudent.Email).
			WithVerificationCode(fixtures.ValidVerificationCode).
			WithStatus(registration.StatusVerified).
			Build()
		s.MockRegistration.SeedRegistration(t, reg)

		err := s.Handler.Handle(t.Context(), StudentComplete{
			Email:            fixtures.TestStudent.Email,
			VerificationCode: fixtures.InvalidVerificationCode,
			Barcode:          fixtures.TestStudent.ID,
			FirstName:        fixtures.TestStudent.FirstName,
			LastName:         fixtures.TestStudent.LastName,
			Password:         fixtures.TestStudent.Password,
			GroupID:          fixtures.TestStudent.GroupID,
		})
		assert.Error(t, err)
		assert.ErrorIs(t, err, registration.ErrInvalidVerificationCode)
	})
}

func TestStudentCompleteHandler_AlreadyCompleted_ShouldFail(t *testing.T) {
	t.Parallel()

	s := NewStudentCompleteSuite(t)
	reg := builders.NewRegistrationBuilder().
		WithEmail(fixtures.ValidStudentEmail).
		WithStatus(registration.StatusCompleted).
		Build()
	s.MockRegistration.SeedRegistration(t, reg)

	err := s.Handler.Handle(t.Context(), StudentComplete{
		Email:            fixtures.TestStudent.Email,
		VerificationCode: reg.VerificationCode(),
		Barcode:          fixtures.TestStudent.ID,
		FirstName:        fixtures.TestStudent.FirstName,
		LastName:         fixtures.TestStudent.LastName,
		Password:         fixtures.TestStudent.Password,
		GroupID:          fixtures.TestStudent.GroupID,
	})
	require.Error(t, err)
	// assert.ErrorIs(t, err, registration.ErrAlreadyCompleted)
}

func TestStudentCompleteHandler_Pending_InvalidVerificationCode_ShouldFail(t *testing.T) {
	t.Parallel()

	s := NewStudentCompleteSuite(t)
	reg := builders.NewRegistrationBuilder().
		WithEmail(fixtures.ValidStudentEmail).
		WithStatus(registration.StatusPending).
		Build()
	s.MockRegistration.SeedRegistration(t, reg)

	err := s.Handler.Handle(t.Context(), StudentComplete{
		Email:            fixtures.TestStudent.Email,
		VerificationCode: fixtures.InvalidVerificationCode,
		Barcode:          fixtures.TestStudent.ID,
		FirstName:        fixtures.TestStudent.FirstName,
		LastName:         fixtures.TestStudent.LastName,
		Password:         fixtures.TestStudent.Password,
		GroupID:          fixtures.TestStudent.GroupID,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, registration.ErrInvalidVerificationCode)
}

func TestStudentCompleteHandler_RegistrationNotFound_ShouldFail(t *testing.T) {
	t.Parallel()

	s := NewStudentCompleteSuite(t)

	err := s.Handler.Handle(t.Context(), StudentComplete{
		Email:            fixtures.TestStudent.Email,
		VerificationCode: fixtures.ValidVerificationCode,
		Barcode:          fixtures.TestStudent.ID,
		FirstName:        fixtures.TestStudent.FirstName,
		LastName:         fixtures.TestStudent.LastName,
		Password:         fixtures.TestStudent.Password,
		GroupID:          fixtures.TestStudent.GroupID,
	})
	require.Error(t, err)
	assert.True(t, errorx.IsNotFound(err), "expected not found error, got: %v", err)
}
