package user_test

import (
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
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
			wantErr: validation.Errors{"id": validation.ErrRequired},
		},
		{
			name:    "missing Email",
			args:    builders.NewStudentBuilder().WithEmail("").BuildRegisterArgs(),
			wantErr: validation.Errors{"email": validation.ErrRequired},
		},
		{
			name:    "missing PassHash",
			args:    builders.NewStudentBuilder().WithPassHash(nil).BuildRegisterArgs(),
			wantErr: validation.Errors{"pass_hash": validation.ErrRequired},
		},
		{
			name:    "empty PassHash",
			args:    builders.NewStudentBuilder().WithPassHash([]byte{}).BuildRegisterArgs(),
			wantErr: validation.Errors{"pass_hash": validation.ErrRequired},
		},
		{
			name:    "missing FirstName",
			args:    builders.NewStudentBuilder().WithFirstName("").BuildRegisterArgs(),
			wantErr: validation.Errors{"first_name": validation.ErrRequired},
		},
		{
			name:    "FirstName too long",
			args:    builders.NewStudentBuilder().WithInvalidLongFirstName().BuildRegisterArgs(),
			wantErr: validation.Errors{"first_name": validation.ErrLengthOutOfRange},
		},
		{
			name:    "FirstName too short",
			args:    builders.NewStudentBuilder().WithInvalidShortFirstName().BuildRegisterArgs(),
			wantErr: validation.Errors{"first_name": validation.ErrLengthOutOfRange},
		},
		{
			name:    "missing LastName",
			args:    builders.NewStudentBuilder().WithLastName("").BuildRegisterArgs(),
			wantErr: validation.Errors{"last_name": validation.ErrRequired},
		},
		{
			name:    "LastName too long",
			args:    builders.NewStudentBuilder().WithInvalidLongLastName().BuildRegisterArgs(),
			wantErr: validation.Errors{"last_name": validation.ErrLengthOutOfRange},
		},
		{
			name:    "LastName too short",
			args:    builders.NewStudentBuilder().WithInvalidShortLastName().BuildRegisterArgs(),
			wantErr: validation.Errors{"last_name": validation.ErrLengthOutOfRange},
		},
		{
			name:    "missing GroupID",
			args:    builders.NewStudentBuilder().WithGroupID(uuid.Nil).BuildRegisterArgs(),
			wantErr: validation.Errors{"group_id": validation.ErrRequired},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			student, err := user.RegisterStudent(tt.args)
			if tt.wantErr == nil {
				user.NewStudentAssertions(student).
					AssertByRegistrationArgs(t, tt.args)
			} else {
				errorx.AssertValidationErrors(t, err, tt.wantErr)
				assert.Nil(t, student, "expected student to be nil on error")
			}
		})
	}
}

func TestRegisterStudent_EmptyArgs(t *testing.T) {
	student, err := user.RegisterStudent(user.RegisterStudentArgs{})
	errorx.AssertValidationErrors(t, err, validation.Errors{
		"id":         validation.ErrRequired,
		"email":      validation.ErrRequired,
		"pass_hash":  validation.ErrRequired,
		"first_name": validation.ErrRequired,
		"last_name":  validation.ErrRequired,
		"group_id":   validation.ErrRequired,
	})
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
			wantErr:    validation.ErrRequired,
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
			wantErr:    validation.ErrRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.student.SetGroupID(tt.newGroupID)
			if tt.wantErr != nil {
				errorx.AssertValidationError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err, "expected no error")
				assert.Equal(t, tt.newGroupID, tt.student.GroupID(), "expected group ID to be updated")
			}
		})
	}
}
