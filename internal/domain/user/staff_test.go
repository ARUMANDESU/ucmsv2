package user_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
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
				NewStaffAssertions(staff).
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

type StaffAssertions struct {
	ID        user.ID
	FirstName string
	LastName  string
	AvatarURL string
	Email     string
	Role      role.Global
	PassHash  []byte
	Events    []event.Event
}

func NewStaffAssertions(s *user.Staff) *StaffAssertions {
	u := s.User()
	return &StaffAssertions{
		ID:        u.ID(),
		FirstName: u.FirstName(),
		LastName:  u.LastName(),
		AvatarURL: u.AvatarUrl(),
		Email:     u.Email(),
		Role:      u.Role(),
		PassHash:  u.PassHash(),
		Events:    s.GetUncommittedEvents(),
	}
}

func (s *StaffAssertions) AssertByRegistrationArgs(t *testing.T, args user.RegisterStaffArgs) *StaffAssertions {
	assert.Equal(t, args.ID, s.ID, "ID mismatch")
	assert.Equal(t, args.FirstName, s.FirstName, "FirstName mismatch")
	assert.Equal(t, args.LastName, s.LastName, "LastName mismatch")
	assert.Equal(t, args.AvatarURL, s.AvatarURL, "AvatarURL mismatch")
	assert.Equal(t, args.Email, s.Email, "Email mismatch")
	assert.Equal(t, role.Staff, s.Role, "Role mismatch")
	assert.Equal(t, args.PassHash, s.PassHash, "PassHash mismatch")

	require.Len(t, s.Events, 1, "expected one event")
	assert.IsType(t, &user.StaffRegistered{}, s.Events[0], "expected StaffRegistered event type")
	staffRegisteredEvent := s.Events[0].(*user.StaffRegistered)
	assert.Equal(t, args.ID, staffRegisteredEvent.StaffID, "StaffID in event mismatch")
	assert.Equal(t, args.Email, staffRegisteredEvent.Email, "Email in event mismatch")
	assert.Equal(t, args.FirstName, staffRegisteredEvent.FirstName, "FirstName in event mismatch")
	assert.Equal(t, args.LastName, staffRegisteredEvent.LastName, "LastName in event mismatch")

	return s
}

func (s *StaffAssertions) AssertID(t *testing.T, expected string) *StaffAssertions {
	assert.Equal(t, expected, s.ID, "ID mismatch")
	return s
}

func (s *StaffAssertions) AssertFirstName(t *testing.T, expected string) *StaffAssertions {
	assert.Equal(t, expected, s.FirstName, "FirstName mismatch")
	return s
}

func (s *StaffAssertions) AssertLastName(t *testing.T, expected string) *StaffAssertions {
	assert.Equal(t, expected, s.LastName, "LastName mismatch")
	return s
}

func (s *StaffAssertions) AssertAvatarURL(t *testing.T, expected string) *StaffAssertions {
	assert.Equal(t, expected, s.AvatarURL, "AvatarURL mismatch")
	return s
}

func (s *StaffAssertions) AssertEmail(t *testing.T, expected string) *StaffAssertions {
	assert.Equal(t, expected, s.Email, "Email mismatch")
	return s
}

func (s *StaffAssertions) AssertRole(t *testing.T, expected role.Global) *StaffAssertions {
	assert.Equal(t, expected, s.Role, "Role mismatch")
	return s
}

func (s *StaffAssertions) AssertPassHash(t *testing.T, expected []byte) *StaffAssertions {
	assert.Equal(t, expected, s.PassHash, "PassHash mismatch")
	return s
}
