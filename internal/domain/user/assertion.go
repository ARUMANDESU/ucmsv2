package user

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
)

type StaffAssertions struct {
	ID        ID
	FirstName string
	LastName  string
	AvatarURL string
	Email     string
	Role      role.Global
	PassHash  []byte
	Events    []event.Event
}

func NewStaffAssertions(s *Staff) *StaffAssertions {
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

func (s *StaffAssertions) AssertByRegistrationArgs(t *testing.T, args RegisterStaffArgs) *StaffAssertions {
	assert.Equal(t, args.ID, s.ID, "ID mismatch")
	assert.Equal(t, args.FirstName, s.FirstName, "FirstName mismatch")
	assert.Equal(t, args.LastName, s.LastName, "LastName mismatch")
	assert.Equal(t, args.AvatarURL, s.AvatarURL, "AvatarURL mismatch")
	assert.Equal(t, args.Email, s.Email, "Email mismatch")
	assert.Equal(t, role.Staff, s.Role, "Role mismatch")
	assert.Equal(t, args.PassHash, s.PassHash, "PassHash mismatch")

	require.Len(t, s.Events, 1, "expected one event")
	assert.IsType(t, &StaffRegistered{}, s.Events[0], "expected StaffRegistered event type")
	staffRegisteredEvent := s.Events[0].(*StaffRegistered)
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

type StudentAssertions struct {
	ID        ID
	FirstName string
	LastName  string
	AvatarURL string
	Email     string
	GroupID   uuid.UUID
	Role      role.Global
	PassHash  []byte
	Events    []event.Event
}

func NewStudentAssertions(s *Student) *StudentAssertions {
	u := s.User()
	return &StudentAssertions{
		ID:        u.ID(),
		FirstName: u.FirstName(),
		LastName:  u.LastName(),
		AvatarURL: u.AvatarUrl(),
		Email:     u.Email(),
		GroupID:   s.GroupID(),
		Role:      u.Role(),
		PassHash:  u.PassHash(),
		Events:    s.GetUncommittedEvents(),
	}
}

func (s *StudentAssertions) AssertByRegistrationArgs(t *testing.T, args RegisterStudentArgs) *StudentAssertions {
	assert.Equal(t, args.ID, s.ID, "ID mismatch")
	assert.Equal(t, args.FirstName, s.FirstName, "FirstName mismatch")
	assert.Equal(t, args.LastName, s.LastName, "LastName mismatch")
	assert.Equal(t, args.AvatarURL, s.AvatarURL, "AvatarURL mismatch")
	assert.Equal(t, args.Email, s.Email, "Email mismatch")
	assert.Equal(t, args.GroupID, s.GroupID, "GroupID mismatch")
	assert.Equal(t, role.Student, s.Role, "Role mismatch")
	assert.Equal(t, args.PassHash, s.PassHash, "PassHash mismatch")

	require.Len(t, s.Events, 1, "expected one event")
	assert.IsType(t, &StudentRegistered{}, s.Events[0], "expected StudentRegistered event type")
	studentRegisteredEvent := s.Events[0].(*StudentRegistered)
	assert.Equal(t, args.ID, studentRegisteredEvent.StudentID, "StudentID in event mismatch")
	assert.Equal(t, args.Email, studentRegisteredEvent.Email, "Email in event mismatch")
	assert.Equal(t, args.FirstName, studentRegisteredEvent.FirstName, "FirstName in event mismatch")
	assert.Equal(t, args.LastName, studentRegisteredEvent.LastName, "LastName in event mismatch")
	assert.Equal(t, args.GroupID, studentRegisteredEvent.GroupID, "GroupID in event mismatch")

	return s
}

func (s *StudentAssertions) AssertID(t *testing.T, expected string) *StudentAssertions {
	assert.Equal(t, expected, s.ID, "ID mismatch")
	return s
}

func (s *StudentAssertions) AssertFirstName(t *testing.T, expected string) *StudentAssertions {
	assert.Equal(t, expected, s.FirstName, "FirstName mismatch")
	return s
}

func (s *StudentAssertions) AssertLastName(t *testing.T, expected string) *StudentAssertions {
	assert.Equal(t, expected, s.LastName, "LastName mismatch")
	return s
}

func (s *StudentAssertions) AssertAvatarURL(t *testing.T, expected string) *StudentAssertions {
	assert.Equal(t, expected, s.AvatarURL, "AvatarURL mismatch")
	return s
}

func (s *StudentAssertions) AssertEmail(t *testing.T, expected string) *StudentAssertions {
	assert.Equal(t, expected, s.Email, "Email mismatch")
	return s
}

func (s *StudentAssertions) AssertGroupID(t *testing.T, expected uuid.UUID) *StudentAssertions {
	assert.Equal(t, expected, s.GroupID, "GroupID mismatch")
	return s
}

func (s *StudentAssertions) AssertRole(t *testing.T, expected role.Global) *StudentAssertions {
	assert.Equal(t, expected, s.Role, "Role mismatch")
	return s
}

func (s *StudentAssertions) AssertPassHash(t *testing.T, expected []byte) *StudentAssertions {
	assert.Equal(t, expected, s.PassHash, "PassHash mismatch")
	return s
}
