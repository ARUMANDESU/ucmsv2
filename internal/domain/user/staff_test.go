package user_test

import (
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/validationx"
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
			name:    "missing Barcode",
			args:    builders.NewStaffBuilder().WithBarcode("").BuildRegisterArgs(),
			wantErr: validation.Errors{"barcode": validation.ErrRequired},
		},
		{
			name:    "missing RegistrationID",
			args:    builders.NewStaffBuilder().WithRegistrationID(registration.ID{}).BuildRegisterArgs(),
			wantErr: validation.Errors{"registration_id": validation.ErrRequired},
		},
		{
			name:    "invalid RegistrationID",
			args:    builders.NewStaffBuilder().WithRegistrationID(registration.ID(uuid.Nil)).BuildRegisterArgs(),
			wantErr: validation.Errors{"registration_id": validation.ErrRequired},
		},
		{
			name:    "missing Email",
			args:    builders.NewStaffBuilder().WithEmail("").BuildRegisterArgs(),
			wantErr: validation.Errors{"email": validation.ErrRequired},
		},
		{
			name:    "invalid Email format",
			args:    builders.NewStaffBuilder().WithEmail("invalid-email").BuildRegisterArgs(),
			wantErr: validation.Errors{"email": is.ErrEmail},
		},
		{
			name:    "missing Password",
			args:    builders.NewStaffBuilder().WithPassword("").BuildRegisterArgs(),
			wantErr: validation.Errors{"password": validation.ErrRequired},
		},
		{
			name:    "invalid Password format",
			args:    builders.NewStaffBuilder().WithPassword("short").BuildRegisterArgs(),
			wantErr: validation.Errors{"password": validation.ErrLengthOutOfRange},
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
				validationx.AssertValidationErrors(t, err, tt.wantErr)
				assert.Nil(t, staff, "expected staff to be nil on error")
			}
		})
	}
}

func TestRegisterStaff_EmptyArgs(t *testing.T) {
	staff, err := user.RegisterStaff(user.RegisterStaffArgs{})
	validationx.AssertValidationErrors(t, err, validation.Errors{
		"barcode":         validation.ErrRequired,
		"registration_id": validation.ErrRequired,
		"email":           validation.ErrRequired,
		"first_name":      validation.ErrRequired,
		"last_name":       validation.ErrRequired,
		"password":        validation.ErrRequired,
	})
	assert.Nil(t, staff, "expected staff to be nil on error")
}
