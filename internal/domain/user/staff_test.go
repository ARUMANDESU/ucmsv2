package user_test

import (
	"testing"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/pkg/validationx"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/builders"
)

func TestAcceptStaffInvitation_ArgValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    user.AcceptStaffInvitationArgs
		wantErr error
	}{
		{
			name:    "valid args",
			args:    builders.NewStaffBuilder().BuildAcceptStaffInvitationArgs(uuid.New()),
			wantErr: nil,
		},
		{
			name:    "missing Barcode",
			args:    builders.NewStaffBuilder().WithBarcode("").BuildAcceptStaffInvitationArgs(uuid.New()),
			wantErr: validation.Errors{"barcode": validation.ErrRequired},
		},
		{
			name:    "missing username",
			args:    builders.NewStaffBuilder().WithUsername("").BuildAcceptStaffInvitationArgs(uuid.New()),
			wantErr: validation.Errors{"username": validation.ErrRequired},
		},
		{
			name:    "username too short",
			args:    builders.NewStaffBuilder().WithUsername("ab").BuildAcceptStaffInvitationArgs(uuid.New()),
			wantErr: validation.Errors{"username": validationx.ErrInvalidUsernameFormat},
		},
		{
			name: "username too long",
			args: builders.NewStaffBuilder().
				WithUsername("a_very_long_username_exceeding_the_maximum_length_of_fifty_characters"). // 69 chars
				BuildAcceptStaffInvitationArgs(uuid.New()),
			wantErr: validation.Errors{"username": validationx.ErrInvalidUsernameFormat},
		},
		{
			name: "username format invalid",
			args: builders.NewStaffBuilder().
				WithUsername("invalid username!"). // contains space and exclamation mark
				BuildAcceptStaffInvitationArgs(uuid.New()),
			wantErr: validation.Errors{"username": validationx.ErrInvalidUsernameFormat},
		},
		{
			name:    "missing Email",
			args:    builders.NewStaffBuilder().WithEmail("").BuildAcceptStaffInvitationArgs(uuid.New()),
			wantErr: validation.Errors{"email": validation.ErrRequired},
		},
		{
			name:    "invalid Email format",
			args:    builders.NewStaffBuilder().WithEmail("invalid-email").BuildAcceptStaffInvitationArgs(uuid.New()),
			wantErr: validation.Errors{"email": is.ErrEmail},
		},
		{
			name:    "missing Password",
			args:    builders.NewStaffBuilder().WithPassword("").BuildAcceptStaffInvitationArgs(uuid.New()),
			wantErr: validation.Errors{"password": validation.ErrRequired},
		},
		{
			name:    "invalid Password format",
			args:    builders.NewStaffBuilder().WithPassword("short").BuildAcceptStaffInvitationArgs(uuid.New()),
			wantErr: validation.Errors{"password": validation.ErrLengthOutOfRange},
		},
		{
			name:    "missing FirstName",
			args:    builders.NewStaffBuilder().WithFirstName("").BuildAcceptStaffInvitationArgs(uuid.New()),
			wantErr: validation.Errors{"first_name": validation.ErrRequired},
		},
		{
			name:    "FirstName too long",
			args:    builders.NewStaffBuilder().WithInvalidLongFirstName().BuildAcceptStaffInvitationArgs(uuid.New()),
			wantErr: validation.Errors{"first_name": validation.ErrLengthOutOfRange},
		},
		{
			name:    "FirstName too short",
			args:    builders.NewStaffBuilder().WithInvalidShortFirstName().BuildAcceptStaffInvitationArgs(uuid.New()),
			wantErr: validation.Errors{"first_name": validation.ErrLengthOutOfRange},
		},
		{
			name:    "missing LastName",
			args:    builders.NewStaffBuilder().WithLastName("").BuildAcceptStaffInvitationArgs(uuid.New()),
			wantErr: validation.Errors{"last_name": validation.ErrRequired},
		},
		{
			name:    "LastName too long",
			args:    builders.NewStaffBuilder().WithInvalidLongLastName().BuildAcceptStaffInvitationArgs(uuid.New()),
			wantErr: validation.Errors{"last_name": validation.ErrLengthOutOfRange},
		},
		{
			name:    "LastName too short",
			args:    builders.NewStaffBuilder().WithInvalidShortLastName().BuildAcceptStaffInvitationArgs(uuid.New()),
			wantErr: validation.Errors{"last_name": validation.ErrLengthOutOfRange},
		},
		{
			name:    "missing InvitationID",
			args:    builders.NewStaffBuilder().BuildAcceptStaffInvitationArgs(uuid.Nil),
			wantErr: validation.Errors{"invitation_id": validation.ErrRequired},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			staff, err := user.AcceptStaffInvitation(tt.args)
			if tt.wantErr == nil {
				user.NewStaffAssertions(staff).
					AssertByAcceptStaffInvitationArgs(t, tt.args)
			} else {
				validationx.AssertValidationErrors(t, err, tt.wantErr)
				assert.Nil(t, staff, "expected staff to be nil on error")
			}
		})
	}
}

func TestRegisterStaff_EmptyArgs(t *testing.T) {
	staff, err := user.AcceptStaffInvitation(user.AcceptStaffInvitationArgs{})
	validationx.AssertValidationErrors(t, err, validation.Errors{
		"barcode":       validation.ErrRequired,
		"username":      validation.ErrRequired,
		"email":         validation.ErrRequired,
		"first_name":    validation.ErrRequired,
		"last_name":     validation.ErrRequired,
		"password":      validation.ErrRequired,
		"invitation_id": validation.ErrRequired,
	})
	assert.Nil(t, staff, "expected staff to be nil on error")
}

func TestCreateInitialStaff_ArgValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    user.CreateInitialStaffArgs
		wantErr error
	}{
		{
			name:    "valid args",
			args:    builders.NewStaffBuilder().BuildCreateInitialStaffArgs(),
			wantErr: nil,
		},
		{
			name:    "missing Barcode",
			args:    builders.NewStaffBuilder().WithBarcode("").BuildCreateInitialStaffArgs(),
			wantErr: validation.Errors{"barcode": validation.ErrRequired},
		},
		{
			name:    "missing username",
			args:    builders.NewStaffBuilder().WithUsername("").BuildCreateInitialStaffArgs(),
			wantErr: validation.Errors{"username": validation.ErrRequired},
		},
		{
			name:    "username too short",
			args:    builders.NewStaffBuilder().WithUsername("ab").BuildCreateInitialStaffArgs(),
			wantErr: validation.Errors{"username": validationx.ErrInvalidUsernameFormat},
		},
		{
			name: "username too long",
			args: builders.NewStaffBuilder().
				WithUsername("a_very_long_username_exceeding_the_maximum_length_of_fifty_characters"). // 69 chars
				BuildCreateInitialStaffArgs(),
			wantErr: validation.Errors{"username": validationx.ErrInvalidUsernameFormat},
		},
		{
			name: "username format invalid",
			args: builders.NewStaffBuilder().
				WithUsername("invalid username!"). // contains space and exclamation mark
				BuildCreateInitialStaffArgs(),
			wantErr: validation.Errors{"username": validationx.ErrInvalidUsernameFormat},
		},
		{
			name:    "missing Email",
			args:    builders.NewStaffBuilder().WithEmail("").BuildCreateInitialStaffArgs(),
			wantErr: validation.Errors{"email": validation.ErrRequired},
		},
		{
			name:    "invalid Email format",
			args:    builders.NewStaffBuilder().WithEmail("invalid-email").BuildCreateInitialStaffArgs(),
			wantErr: validation.Errors{"email": is.ErrEmail},
		},
		{
			name:    "missing FirstName",
			args:    builders.NewStaffBuilder().WithFirstName("").BuildCreateInitialStaffArgs(),
			wantErr: validation.Errors{"first_name": validation.ErrRequired},
		},
		{
			name:    "FirstName too long",
			args:    builders.NewStaffBuilder().WithInvalidLongFirstName().BuildCreateInitialStaffArgs(),
			wantErr: validation.Errors{"first_name": validation.ErrLengthOutOfRange},
		},
		{
			name:    "FirstName too short",
			args:    builders.NewStaffBuilder().WithInvalidShortFirstName().BuildCreateInitialStaffArgs(),
			wantErr: validation.Errors{"first_name": validation.ErrLengthOutOfRange},
		},
		{
			name:    "missing LastName",
			args:    builders.NewStaffBuilder().WithLastName("").BuildCreateInitialStaffArgs(),
			wantErr: validation.Errors{"last_name": validation.ErrRequired},
		},
		{
			name:    "LastName too long",
			args:    builders.NewStaffBuilder().WithInvalidLongLastName().BuildCreateInitialStaffArgs(),
			wantErr: validation.Errors{"last_name": validation.ErrLengthOutOfRange},
		},
		{
			name:    "LastName too short",
			args:    builders.NewStaffBuilder().WithInvalidShortLastName().BuildCreateInitialStaffArgs(),
			wantErr: validation.Errors{"last_name": validation.ErrLengthOutOfRange},
		},
		{
			name:    "missing Password",
			args:    builders.NewStaffBuilder().WithPassword("").BuildCreateInitialStaffArgs(),
			wantErr: validation.Errors{"password": validation.ErrRequired},
		},
		{
			name:    "invalid Password format",
			args:    builders.NewStaffBuilder().WithPassword("short").BuildCreateInitialStaffArgs(),
			wantErr: validation.Errors{"password": validation.ErrLengthOutOfRange},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			staff, err := user.CreateInitialStaff(tt.args)
			if tt.wantErr == nil {
				user.NewStaffAssertions(staff).
					AssertByCreateInitialArgs(t, tt.args)
			} else {
				validationx.AssertValidationErrors(t, err, tt.wantErr)
				assert.Nil(t, staff, "expected staff to be nil on error")
			}
		})
	}
}

func TestCreateInitialStaff_EmptyArgs(t *testing.T) {
	staff, err := user.CreateInitialStaff(user.CreateInitialStaffArgs{})
	validationx.AssertValidationErrors(t, err, validation.Errors{
		"barcode":    validation.ErrRequired,
		"username":   validation.ErrRequired,
		"email":      validation.ErrRequired,
		"first_name": validation.ErrRequired,
		"last_name":  validation.ErrRequired,
		"password":   validation.ErrRequired,
	})
	assert.Nil(t, staff, "expected staff to be nil on error")
}
