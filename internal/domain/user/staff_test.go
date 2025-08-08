package user_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ARUMANDESU/ucms/internal/domain/user"
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
			wantErr: user.ErrMissingID,
		},
		{
			name:    "missing Email",
			args:    builders.NewStaffBuilder().WithEmail("").BuildRegisterArgs(),
			wantErr: user.ErrMissingEmail,
		},
		{
			name:    "missing PassHash",
			args:    builders.NewStaffBuilder().WithPassHash(nil).BuildRegisterArgs(),
			wantErr: user.ErrMissingPassHash,
		},
		{
			name:    "empty PassHash",
			args:    builders.NewStaffBuilder().WithPassHash([]byte{}).BuildRegisterArgs(),
			wantErr: user.ErrMissingPassHash,
		},
		{
			name:    "missing FirstName",
			args:    builders.NewStaffBuilder().WithFirstName("").BuildRegisterArgs(),
			wantErr: user.ErrMissingFirstName,
		},
		{
			name:    "FirstName too long",
			args:    builders.NewStaffBuilder().WithInvalidLongFirstName().BuildRegisterArgs(),
			wantErr: user.ErrFirstNameTooLong,
		},
		{
			name:    "FirstName too short",
			args:    builders.NewStaffBuilder().WithInvalidShortFirstName().BuildRegisterArgs(),
			wantErr: user.ErrFirstNameTooShort,
		},
		{
			name:    "missing LastName",
			args:    builders.NewStaffBuilder().WithLastName("").BuildRegisterArgs(),
			wantErr: user.ErrMissingLastName,
		},
		{
			name:    "LastName too long",
			args:    builders.NewStaffBuilder().WithInvalidLongLastName().BuildRegisterArgs(),
			wantErr: user.ErrLastNameTooLong,
		},
		{
			name:    "LastName too short",
			args:    builders.NewStaffBuilder().WithInvalidShortLastName().BuildRegisterArgs(),
			wantErr: user.ErrLastNameTooShort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			staff, err := user.RegisterStaff(tt.args)
			if tt.wantErr == nil {
				user.NewStaffAssertions(staff).
					AssertByRegistrationArgs(t, tt.args)
			} else {
				assert.ErrorIs(t, err, tt.wantErr, "expected error %v, got %v", tt.wantErr, err)
				assert.Nil(t, staff, "expected staff to be nil on error")
			}
		})
	}
}

func TestRegisterStaff_EmptyArgs(t *testing.T) {
	staff, err := user.RegisterStaff(user.RegisterStaffArgs{})
	assert.ErrorIs(t, err, user.ErrMissingID, "expected ErrMissingID for empty args")
	assert.Nil(t, staff, "expected staff to be nil on error")
}
