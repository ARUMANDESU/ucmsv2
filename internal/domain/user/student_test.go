package user_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/tests/integration/builders"
)

func TestRegisterStudent_ArgValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    user.RegisterStudentArgs
		wantErr error
	}{
		{
			name:    "valid args",
			args:    builders.NewStudentBuilder().BuildRegisterArgs(),
			wantErr: nil,
		},
		{
			name:    "missing ID",
			args:    builders.NewStudentBuilder().WithID("").BuildRegisterArgs(),
			wantErr: user.ErrMissingID,
		},
		{
			name:    "missing Email",
			args:    builders.NewStudentBuilder().WithEmail("").BuildRegisterArgs(),
			wantErr: user.ErrMissingEmail,
		},
		{
			name:    "missing PassHash",
			args:    builders.NewStudentBuilder().WithPassHash(nil).BuildRegisterArgs(),
			wantErr: user.ErrMissingPassHash,
		},
		{
			name:    "empty PassHash",
			args:    builders.NewStudentBuilder().WithPassHash([]byte{}).BuildRegisterArgs(),
			wantErr: user.ErrMissingPassHash,
		},
		{
			name:    "missing FirstName",
			args:    builders.NewStudentBuilder().WithFirstName("").BuildRegisterArgs(),
			wantErr: user.ErrMissingFirstName,
		},
		{
			name:    "FirstName too long",
			args:    builders.NewStudentBuilder().WithInvalidLongFirstName().BuildRegisterArgs(),
			wantErr: user.ErrFirstNameTooLong,
		},
		{
			name:    "FirstName too short",
			args:    builders.NewStudentBuilder().WithInvalidShortFirstName().BuildRegisterArgs(),
			wantErr: user.ErrFirstNameTooShort,
		},
		{
			name:    "missing LastName",
			args:    builders.NewStudentBuilder().WithLastName("").BuildRegisterArgs(),
			wantErr: user.ErrMissingLastName,
		},
		{
			name:    "LastName too long",
			args:    builders.NewStudentBuilder().WithInvalidLongLastName().BuildRegisterArgs(),
			wantErr: user.ErrLastNameTooLong,
		},
		{
			name:    "LastName too short",
			args:    builders.NewStudentBuilder().WithInvalidShortLastName().BuildRegisterArgs(),
			wantErr: user.ErrLastNameTooShort,
		},
		{
			name:    "missing GroupID",
			args:    builders.NewStudentBuilder().WithGroupID(uuid.Nil).BuildRegisterArgs(),
			wantErr: user.ErrMissingGroupID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			student, err := user.RegisterStudent(tt.args)
			if tt.wantErr == nil {
				user.NewStudentAssertions(student).
					AssertByRegistrationArgs(t, tt.args)
			} else {
				assert.ErrorIs(t, err, tt.wantErr, "expected error %v, got %v", tt.wantErr, err)
				assert.Nil(t, student, "expected student to be nil on error")
			}
		})
	}
}

func TestRegisterStudent_EmptyArgs(t *testing.T) {
	student, err := user.RegisterStudent(user.RegisterStudentArgs{})
	assert.ErrorIs(t, err, user.ErrMissingID, "expected ErrMissingID for empty args")
	assert.Nil(t, student, "expected student to be nil on error")
}

func TestStudent_SetGroupID(t *testing.T) {
	tests := []struct {
		name       string
		student    *user.Student
		newGroupID uuid.UUID
		wantErr    error
	}{
		{
			name:       "given valid group ID",
			student:    builders.NewStudentBuilder().Build(),
			newGroupID: uuid.New(),
		},
		{
			name:       "given nil group ID",
			student:    builders.NewStudentBuilder().Build(),
			newGroupID: uuid.Nil,
			wantErr:    user.ErrMissingGroupID,
		},
		{
			name:       "given same group ID",
			student:    builders.NewStudentBuilder().Build(),
			newGroupID: builders.NewStudentBuilder().Build().GroupID(),
			wantErr:    nil, // No error expected when setting the same group ID
		},
		{
			name:       "given empty group ID",
			student:    builders.NewStudentBuilder().Build(),
			newGroupID: uuid.UUID{},
			wantErr:    user.ErrMissingGroupID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.student.SetGroupID(tt.newGroupID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr, "expected error %v, got %v", tt.wantErr, err)
			} else {
				require.NoError(t, err, "expected no error")
				assert.Equal(t, tt.newGroupID, tt.student.GroupID(), "expected group ID to be updated")
			}
		})
	}
}
