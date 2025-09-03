package cmd

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/internal/domain/group"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
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
	MockStudent      *mocks.StudentRepo
}

func NewStudentCompleteSuite(t *testing.T) *StudentCompleteSuite {
	mockUser := mocks.NewUserRepo()
	mockRegistration := mocks.NewRegistrationRepo()
	mockGroup := mocks.NewGroupRepo()
	mockStudent := mocks.NewStudentRepo()

	// Seed a group for the test
	group := builders.NewGroupBuilder().Build()
	mockGroup.SeedGroup(t, group)

	handler := NewStudentCompleteHandler(StudentCompleteHandlerArgs{
		UserGetter:       mockUser,
		RegistrationRepo: mockRegistration,
		GroupGetter:      mockGroup,
		StudentSaver:     mockStudent,
	})

	return &StudentCompleteSuite{
		Handler:          handler,
		MockUser:         mockUser,
		MockRegistration: mockRegistration,
		MockGroup:        mockGroup,
		MockStudent:      mockStudent,
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
			Barcode:          fixtures.TestStudent.Barcode,
			Username:         fixtures.TestStudent.Username,
			FirstName:        fixtures.TestStudent.FirstName,
			LastName:         fixtures.TestStudent.LastName,
			Password:         fixtures.TestStudent.Password,
			GroupID:          fixtures.TestStudent.GroupID,
		})
		require.NoError(t, err)

		s.MockStudent.RequireStudentByBarcode(t, user.Barcode(fixtures.TestStudent.Barcode)).
			AssertEmail(t, fixtures.ValidStudentEmail).
			AssertAvatarURL(t, "").
			AssertUsername(t, fixtures.TestStudent.Username).
			AssertFirstName(t, fixtures.TestStudent.FirstName).
			AssertLastName(t, fixtures.TestStudent.LastName).
			AssertGroupID(t, fixtures.TestStudent.GroupID).
			AssertPassword(t, fixtures.TestStudent.Password).
			AssertRole(t, role.Student)

		s.MockStudent.AssertEventCount(t, 1)
		e := mocks.RequireEventExists(t, s.MockStudent.EventRepo, &user.StudentRegistered{})
		user.NewStudentRegistrationAssertions(t, e).
			AssertStudentBarcode(user.Barcode(fixtures.TestStudent.Barcode)).
			AssertStudentUsername(fixtures.TestStudent.Username).
			AssertRegistrationID(reg.ID()).
			AssertEmail(fixtures.TestStudent.Email).
			AssertFirstName(fixtures.TestStudent.FirstName).
			AssertLastName(fixtures.TestStudent.LastName).
			AssertGroupID(fixtures.TestStudent.GroupID)
	})
}

func TestStudentCompleteHandler_UserAlreadyExists_ShouldFail(t *testing.T) {
	t.Parallel()

	t.Run("not verified yet", func(t *testing.T) {
		s := NewStudentCompleteSuite(t)
		reg := builders.NewRegistrationBuilder().
			WithEmail(fixtures.ValidStudentEmail).
			WithStatus(registration.StatusPending).
			Build()
		s.MockRegistration.SeedRegistration(t, reg)
		err := s.Handler.Handle(t.Context(), StudentComplete{
			Email:            fixtures.TestStudent.Email,
			VerificationCode: reg.VerificationCode(),
			Barcode:          fixtures.TestStudent.Barcode,
			Username:         fixtures.TestStudent.Username,
			FirstName:        fixtures.TestStudent.FirstName,
			LastName:         fixtures.TestStudent.LastName,
			Password:         fixtures.TestStudent.Password,
			GroupID:          fixtures.TestStudent.GroupID,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, registration.ErrVerifyFirst)
	})

	t.Run("user by email already exists", func(t *testing.T) {
		s := NewStudentCompleteSuite(t)
		u := builders.NewStudentBuilder().Build().User()
		s.MockUser.SeedUser(t, u)

		err := s.Handler.Handle(t.Context(), StudentComplete{
			Email:            u.Email(),
			VerificationCode: fixtures.ValidVerificationCode,
			Barcode:          u.Barcode(),
			Username:         u.Username(),
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
			Barcode:          u.Barcode(),
			Username:         fixtures.TestStudent.Username,
			FirstName:        fixtures.TestStudent.FirstName,
			LastName:         fixtures.TestStudent.LastName,
			Password:         fixtures.TestStudent.Password,
			GroupID:          fixtures.TestStudent.GroupID,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrBarcodeNotAvailable)
	})
	t.Run("user by email, username and barcode already exists", func(t *testing.T) {
		s := NewStudentCompleteSuite(t)
		u := builders.NewStudentBuilder().Build().User()
		s.MockUser.SeedUser(t, u)

		err := s.Handler.Handle(t.Context(), StudentComplete{
			Email:            u.Email(),
			VerificationCode: fixtures.ValidVerificationCode,
			Barcode:          u.Barcode(),
			Username:         u.Username(),
			FirstName:        fixtures.TestStudent.FirstName,
			LastName:         fixtures.TestStudent.LastName,
			Password:         fixtures.TestStudent.Password,
			GroupID:          fixtures.TestStudent.GroupID,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmailNotAvailable)
		assert.ErrorIs(t, err, ErrBarcodeNotAvailable)
		assert.ErrorIs(t, err, ErrUsernameNotAvailable)
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
			Barcode:          fixtures.TestStudent.Barcode,
			Username:         fixtures.TestStudent.Username,
			FirstName:        fixtures.TestStudent.FirstName,
			LastName:         fixtures.TestStudent.LastName,
			Password:         fixtures.TestStudent.Password,
			GroupID:          group.ID(uuid.New()),
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
			Barcode:          fixtures.TestStudent.Barcode,
			Username:         fixtures.TestStudent.Username,
			FirstName:        fixtures.TestStudent.FirstName,
			LastName:         fixtures.TestStudent.LastName,
			Password:         fixtures.TestStudent.Password,
			GroupID:          fixtures.TestStudent.GroupID,
		})
		assert.Error(t, err)
		assert.ErrorIs(t, err, registration.ErrInvalidVerificationCode, "expected invalid verification code error, got: %v", err)
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
		Barcode:          fixtures.TestStudent.Barcode,
		Username:         fixtures.TestStudent.Username,
		FirstName:        fixtures.TestStudent.FirstName,
		LastName:         fixtures.TestStudent.LastName,
		Password:         fixtures.TestStudent.Password,
		GroupID:          fixtures.TestStudent.GroupID,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, registration.ErrRegistrationCompleted)
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
		Barcode:          fixtures.TestStudent.Barcode,
		Username:         fixtures.TestStudent.Username,
		FirstName:        fixtures.TestStudent.FirstName,
		LastName:         fixtures.TestStudent.LastName,
		Password:         fixtures.TestStudent.Password,
		GroupID:          fixtures.TestStudent.GroupID,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, registration.ErrVerifyFirst)
}

func TestStudentCompleteHandler_RegistrationNotFound_ShouldFail(t *testing.T) {
	t.Parallel()

	s := NewStudentCompleteSuite(t)

	err := s.Handler.Handle(t.Context(), StudentComplete{
		Email:            fixtures.TestStudent.Email,
		VerificationCode: fixtures.ValidVerificationCode,
		Barcode:          fixtures.TestStudent.Barcode,
		Username:         fixtures.TestStudent.Username,
		FirstName:        fixtures.TestStudent.FirstName,
		LastName:         fixtures.TestStudent.LastName,
		Password:         fixtures.TestStudent.Password,
		GroupID:          fixtures.TestStudent.GroupID,
	})
	require.Error(t, err)
	assert.True(t, errorx.IsNotFound(err), "expected not found error, got: %v", err)
}
