package user_test

import (
	"testing"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/internal/domain/group"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/validationx"
	"github.com/ARUMANDESU/ucms/tests/integration/builders"
)

func TestRegisterStudent_ArgValidation(t *testing.T) {
	t.Parallel()

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
			name:    "missing barcode",
			args:    builders.NewStudentBuilder().WithBarcode("").BuildRegisterArgs(),
			wantErr: validation.Errors{"barcode": validation.ErrRequired},
		},
		{
			name:    "missing username",
			args:    builders.NewStudentBuilder().WithUsername("").BuildRegisterArgs(),
			wantErr: validation.Errors{"username": validation.ErrRequired},
		},
		{
			name:    "username too short",
			args:    builders.NewStudentBuilder().WithUsername("ab").BuildRegisterArgs(),
			wantErr: validation.Errors{"username": validationx.ErrInvalidUsernameFormat},
		},
		{
			name: "username too long",
			args: builders.NewStudentBuilder().
				WithUsername("a_very_long_username_exceeding_the_maximum_length_of_fifty_characters"). // 69 chars
				BuildRegisterArgs(),
			wantErr: validation.Errors{"username": validationx.ErrInvalidUsernameFormat},
		},
		{
			name: "username format invalid",
			args: builders.NewStudentBuilder().
				WithUsername("invalid username!"). // contains space and exclamation mark
				BuildRegisterArgs(),
			wantErr: validation.Errors{"username": validationx.ErrInvalidUsernameFormat},
		},
		{
			name:    "missing Email",
			args:    builders.NewStudentBuilder().WithEmail("").BuildRegisterArgs(),
			wantErr: validation.Errors{"email": validation.ErrRequired},
		},
		{
			name:    "invalid Email format",
			args:    builders.NewStudentBuilder().WithEmail("invalid-email").BuildRegisterArgs(),
			wantErr: validation.Errors{"email": is.ErrEmail},
		},
		{
			name:    "missing RegistrationID",
			args:    builders.NewStudentBuilder().WithRegistrationID(registration.ID(uuid.Nil)).BuildRegisterArgs(),
			wantErr: validation.Errors{"registration_id": validation.ErrRequired},
		},
		{
			name:    "missing Password",
			args:    builders.NewStudentBuilder().WithPassword("").BuildRegisterArgs(),
			wantErr: validation.Errors{"password": validation.ErrRequired},
		},
		{
			name:    "invalid Password format",
			args:    builders.NewStudentBuilder().WithPassword("short").BuildRegisterArgs(),
			wantErr: validation.Errors{"password": validation.ErrLengthOutOfRange},
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
			args:    builders.NewStudentBuilder().WithGroupID(group.ID(uuid.Nil)).BuildRegisterArgs(),
			wantErr: validation.Errors{"group_id": validation.ErrRequired},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			student, err := user.RegisterStudent(tt.args)
			if tt.wantErr == nil {
				require.NoError(t, err, "expected no error for valid args")
				user.NewStudentAssertions(student).
					AssertByRegistrationArgs(t, tt.args)
			} else {
				validationx.AssertValidationErrors(t, err, tt.wantErr)
				assert.Nil(t, student, "expected student to be nil on error")
			}
		})
	}
}

func TestRegisterStudent_EmptyArgs(t *testing.T) {
	student, err := user.RegisterStudent(user.RegisterStudentArgs{})
	validationx.AssertValidationErrors(t, err, validation.Errors{
		"barcode":         validation.ErrRequired,
		"username":        validation.ErrRequired,
		"registration_id": validation.ErrRequired,
		"email":           validation.ErrRequired,
		"password":        validation.ErrRequired,
		"first_name":      validation.ErrRequired,
		"last_name":       validation.ErrRequired,
		"group_id":        validation.ErrRequired,
	})
	assert.Nil(t, student, "expected student to be nil on error")
}

func TestStudent_SetGroupID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		student    *user.Student
		newGroupID group.ID
		wantErr    error
	}{
		{
			name:       "given valid group ID",
			student:    builders.NewStudentBuilder().Build(),
			newGroupID: group.ID(uuid.New()),
		},
		{
			name:       "given nil group ID",
			student:    builders.NewStudentBuilder().Build(),
			newGroupID: group.ID(uuid.Nil),
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
			newGroupID: group.ID{},
			wantErr:    validation.ErrRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.student.SetGroupID(tt.newGroupID)
			if tt.wantErr != nil {
				validationx.AssertValidationError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err, "expected no error")
				assert.Equal(t, tt.newGroupID, tt.student.GroupID(), "expected group ID to be updated")
			}
		})
	}
}
