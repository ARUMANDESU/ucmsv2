package user_test

import (
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/stretchr/testify/assert"

	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/tests/integration/builders"
)

func TestRegisterStaff_ArgValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    user.RegisterStaffArgs
		wantErr error
	}{
		{
			name:    "valid args",
			args:    builders.NewStaffBuilder().BuildRegisterArgs(),
			wantErr: nil,
		},
		{
			name:    "missing ID",
			args:    builders.NewStaffBuilder().WithID("").BuildRegisterArgs(),
			wantErr: validation.Errors{"id": validation.ErrRequired},
		},
		{
			name:    "missing Email",
			args:    builders.NewStaffBuilder().WithEmail("").BuildRegisterArgs(),
			wantErr: validation.Errors{"email": validation.ErrRequired},
		},
		{
			name:    "missing PassHash",
			args:    builders.NewStaffBuilder().WithPassHash(nil).BuildRegisterArgs(),
			wantErr: validation.Errors{"pass_hash": validation.ErrRequired},
		},
		{
			name:    "empty PassHash",
			args:    builders.NewStaffBuilder().WithPassHash([]byte{}).BuildRegisterArgs(),
			wantErr: validation.Errors{"pass_hash": validation.ErrRequired},
		},
		{
			name:    "missing FirstName",
			args:    builders.NewStaffBuilder().WithFirstName("").BuildRegisterArgs(),
			wantErr: validation.Errors{"first_name": validation.ErrRequired},
		},
		{
			name:    "FirstName too long",
			args:    builders.NewStaffBuilder().WithInvalidLongFirstName().BuildRegisterArgs(),
			wantErr: validation.Errors{"first_name": validation.ErrLengthOutOfRange},
		},
		{
			name:    "FirstName too short",
			args:    builders.NewStaffBuilder().WithInvalidShortFirstName().BuildRegisterArgs(),
			wantErr: validation.Errors{"first_name": validation.ErrLengthOutOfRange},
		},
		{
			name:    "missing LastName",
			args:    builders.NewStaffBuilder().WithLastName("").BuildRegisterArgs(),
			wantErr: validation.Errors{"last_name": validation.ErrRequired},
		},
		{
			name:    "LastName too long",
			args:    builders.NewStaffBuilder().WithInvalidLongLastName().BuildRegisterArgs(),
			wantErr: validation.Errors{"last_name": validation.ErrLengthOutOfRange},
		},
		{
			name:    "LastName too short",
			args:    builders.NewStaffBuilder().WithInvalidShortLastName().BuildRegisterArgs(),
			wantErr: validation.Errors{"last_name": validation.ErrLengthOutOfRange},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			staff, err := user.RegisterStaff(tt.args)
			if tt.wantErr == nil {
				user.NewStaffAssertions(staff).
					AssertByRegistrationArgs(t, tt.args)
			} else {
				errorx.AssertValidationErrors(t, err, tt.wantErr)
				assert.Nil(t, staff, "expected staff to be nil on error")
			}
		})
	}
}

func TestRegisterStaff_EmptyArgs(t *testing.T) {
	staff, err := user.RegisterStaff(user.RegisterStaffArgs{})
	errorx.AssertValidationErrors(t, err, validation.Errors{
		"id":         validation.ErrRequired,
		"email":      validation.ErrRequired,
		"first_name": validation.ErrRequired,
		"last_name":  validation.ErrRequired,
		"pass_hash":  validation.ErrRequired,
	})
	assert.Nil(t, staff, "expected staff to be nil on error")
}
