package event

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/tests/integration/fixtures"
	"github.com/ARUMANDESU/ucms/tests/mocks"
)

type StudentRegistrationCompletedSuite struct {
	Handler         *StudentRegistrationCompletedHandler
	MockStudentRepo *mocks.StudentRepo
}

func NewStudentRegistrationCompletedSuite() *StudentRegistrationCompletedSuite {
	mockStudentRepo := mocks.NewStudentRepo()
	handler := NewStudentRegistrationCompletedHandler(StudentRegistrationCompletedHandlerArgs{
		StudentRepo: mockStudentRepo,
	})

	return &StudentRegistrationCompletedSuite{
		Handler:         handler,
		MockStudentRepo: mockStudentRepo,
	}
}

func TestStudentRegistrationCompletedHandler_HappyPath(t *testing.T) {
	t.Parallel()

	s := NewStudentRegistrationCompletedSuite()

	event := &registration.StudentRegistrationCompleted{
		RegistrationID: registration.ID(uuid.New()),
		Barcode:        fixtures.TestStudent.ID,
		Email:          fixtures.TestStudent.Email,
		FirstName:      fixtures.TestStudent.FirstName,
		LastName:       fixtures.TestStudent.LastName,
		PassHash:       []byte("hashedpassword"),
		GroupID:        fixtures.TestStudent.GroupID,
	}

	err := s.Handler.Handle(context.Background(), event)
	require.NoError(t, err)

	student := s.MockStudentRepo.RequireStudentByID(t, user.ID(fixtures.TestStudent.ID))
	student.
		AssertID(t, user.ID(fixtures.TestStudent.ID)).
		AssertEmail(t, fixtures.TestStudent.Email).
		AssertFirstName(t, fixtures.TestStudent.FirstName).
		AssertLastName(t, fixtures.TestStudent.LastName).
		AssertPassHash(t, []byte("hashedpassword")).
		AssertRole(t, role.Student).
		AssertGroupID(t, fixtures.TestStudent.GroupID)
}

func TestStudentRegistrationCompletedHandler_NilEvent(t *testing.T) {
	t.Parallel()

	s := NewStudentRegistrationCompletedSuite()

	err := s.Handler.Handle(context.Background(), nil)
	assert.NoError(t, err)

	s.MockStudentRepo.AssertStudentNotExistsByID(t, user.ID(fixtures.TestStudent.ID))
}

func TestStudentRegistrationCompletedHandler_RegisterStudentError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event *registration.StudentRegistrationCompleted
		want  error
	}{
		{
			name: "missing ID",
			event: &registration.StudentRegistrationCompleted{
				RegistrationID: registration.ID(uuid.New()),
				Barcode:        "",
				Email:          fixtures.TestStudent.Email,
				FirstName:      fixtures.TestStudent.FirstName,
				LastName:       fixtures.TestStudent.LastName,
				PassHash:       []byte("hashedpassword"),
				GroupID:        fixtures.TestStudent.GroupID,
			},
			want: user.ErrMissingID,
		},
		{
			name: "missing email",
			event: &registration.StudentRegistrationCompleted{
				RegistrationID: registration.ID(uuid.New()),
				Barcode:        fixtures.TestStudent.ID,
				Email:          "",
				FirstName:      fixtures.TestStudent.FirstName,
				LastName:       fixtures.TestStudent.LastName,
				PassHash:       []byte("hashedpassword"),
				GroupID:        fixtures.TestStudent.GroupID,
			},
			want: user.ErrMissingEmail,
		},
		{
			name: "missing first name",
			event: &registration.StudentRegistrationCompleted{
				RegistrationID: registration.ID(uuid.New()),
				Barcode:        fixtures.TestStudent.ID,
				Email:          fixtures.TestStudent.Email,
				FirstName:      "",
				LastName:       fixtures.TestStudent.LastName,
				PassHash:       []byte("hashedpassword"),
				GroupID:        fixtures.TestStudent.GroupID,
			},
			want: user.ErrMissingFirstName,
		},
		{
			name: "missing last name",
			event: &registration.StudentRegistrationCompleted{
				RegistrationID: registration.ID(uuid.New()),
				Barcode:        fixtures.TestStudent.ID,
				Email:          fixtures.TestStudent.Email,
				FirstName:      fixtures.TestStudent.FirstName,
				LastName:       "",
				PassHash:       []byte("hashedpassword"),
				GroupID:        fixtures.TestStudent.GroupID,
			},
			want: user.ErrMissingLastName,
		},
		{
			name: "missing group ID",
			event: &registration.StudentRegistrationCompleted{
				RegistrationID: registration.ID(uuid.New()),
				Barcode:        fixtures.TestStudent.ID,
				Email:          fixtures.TestStudent.Email,
				FirstName:      fixtures.TestStudent.FirstName,
				LastName:       fixtures.TestStudent.LastName,
				PassHash:       []byte("hashedpassword"),
				GroupID:        uuid.Nil,
			},
			want: user.ErrMissingGroupID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStudentRegistrationCompletedSuite()

			err := s.Handler.Handle(context.Background(), tt.event)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.want)

			s.MockStudentRepo.AssertStudentNotExistsByID(t, user.ID(fixtures.TestStudent.ID))
		})
	}
}

func TestStudentRegistrationCompletedHandler_SaveStudentError(t *testing.T) {
	t.Parallel()

	t.Run("student already exists", func(t *testing.T) {
		s := NewStudentRegistrationCompletedSuite()

		existingStudent, err := user.RegisterStudent(user.RegisterStudentArgs{
			ID:        user.ID(fixtures.TestStudent.ID),
			FirstName: fixtures.TestStudent.FirstName,
			LastName:  fixtures.TestStudent.LastName,
			Email:     fixtures.TestStudent.Email,
			PassHash:  []byte("existingpasshash"),
			GroupID:   fixtures.TestStudent.GroupID,
		})
		require.NoError(t, err)

		s.MockStudentRepo.SeedStudent(t, existingStudent)

		event := &registration.StudentRegistrationCompleted{
			RegistrationID: registration.ID(uuid.New()),
			Barcode:        fixtures.TestStudent.ID,
			Email:          fixtures.TestStudent.Email,
			FirstName:      fixtures.TestStudent.FirstName,
			LastName:       fixtures.TestStudent.LastName,
			PassHash:       []byte("newpasshash"),
			GroupID:        fixtures.TestStudent.GroupID,
		}

		err = s.Handler.Handle(context.Background(), event)
		require.Error(t, err)
		assert.True(t, errorx.IsDuplicateEntry(err), "expected duplicate entry error, got: %v", err)
	})
}
