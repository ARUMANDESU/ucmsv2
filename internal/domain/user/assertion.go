package user

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/ARUMANDESU/ucms/internal/domain/group"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
)

type UserAssertions struct {
	t    *testing.T
	user *User
}

func NewUserAssertions(t *testing.T, u *User) *UserAssertions {
	return &UserAssertions{t: t, user: u}
}

func (u *UserAssertions) AssertIDNotEmpty() *UserAssertions {
	u.t.Helper()
	assert.NotEmpty(u.t, u.user.id, "ID should not be empty")
	return u
}

func (u *UserAssertions) AssertID(expected ID) *UserAssertions {
	u.t.Helper()
	assert.Equal(u.t, expected, u.user.id, "ID mismatch")
	return u
}

func (u *UserAssertions) AssertBarcode(expected Barcode) *UserAssertions {
	u.t.Helper()
	assert.Equal(u.t, expected, u.user.barcode, "Barcode mismatch")
	return u
}

func (u *UserAssertions) AssertUsername(expected string) *UserAssertions {
	u.t.Helper()
	assert.Equal(u.t, expected, u.user.username, "Username mismatch")
	return u
}

func (u *UserAssertions) AssertFirstName(expected string) *UserAssertions {
	u.t.Helper()
	assert.Equal(u.t, expected, u.user.firstName, "FirstName mismatch")
	return u
}

func (u *UserAssertions) AssertLastName(expected string) *UserAssertions {
	u.t.Helper()
	assert.Equal(u.t, expected, u.user.lastName, "LastName mismatch")
	return u
}

func (u *UserAssertions) AssertAvatarURL(expected string) *UserAssertions {
	u.t.Helper()
	assert.Equal(u.t, expected, u.user.avatarURL, "AvatarURL mismatch")
	return u
}

func (u *UserAssertions) AssertEmail(expected string) *UserAssertions {
	u.t.Helper()
	assert.Equal(u.t, expected, u.user.email, "Email mismatch")
	return u
}

func (u *UserAssertions) AssertRole(expected role.Global) *UserAssertions {
	u.t.Helper()
	assert.Equal(u.t, expected, u.user.role, "Role mismatch")
	return u
}

func (u *UserAssertions) AssertPassword(expected string) *UserAssertions {
	u.t.Helper()
	err := bcrypt.CompareHashAndPassword(u.user.passHash, []byte(expected))
	assert.NoError(u.t, err, "PassHash mismatch")
	return u
}

func (u *UserAssertions) AssertCreatedAtWithin(expected time.Time, delta time.Duration) *UserAssertions {
	u.t.Helper()
	assert.WithinDuration(u.t, expected, u.user.createdAt, delta, "CreatedAt mismatch")
	return u
}

func (u *UserAssertions) AssertUpdatedAtWithin(expected time.Time, delta time.Duration) *UserAssertions {
	u.t.Helper()
	assert.WithinDuration(u.t, expected, u.user.updatedAt, delta, "UpdatedAt mismatch")
	return u
}

type StaffAssertions struct {
	staff *Staff
}

func NewStaffAssertions(s *Staff) *StaffAssertions {
	return &StaffAssertions{
		staff: s,
	}
}

func (s *StaffAssertions) AssertByAcceptStaffInvitationArgs(t *testing.T, args AcceptStaffInvitationArgs) *StaffAssertions {
	t.Helper()
	assert.NotEmpty(t, s.staff.user.id, "ID should not be empty")
	assert.Equal(t, args.Barcode, s.staff.user.barcode, "Barcode mismatch")
	assert.Equal(t, args.Username, s.staff.user.username, "Username mismatch")
	assert.Equal(t, args.FirstName, s.staff.user.firstName, "FirstName mismatch")
	assert.Equal(t, args.LastName, s.staff.user.lastName, "LastName mismatch")
	assert.Equal(t, args.Email, s.staff.user.email, "Email mismatch")
	assert.Equal(t, role.Staff, s.staff.user.role, "Role mismatch")
	assert.WithinDuration(t, time.Now(), s.staff.user.createdAt, time.Minute, "CreatedAt should be recent")
	assert.WithinDuration(t, time.Now(), s.staff.user.updatedAt, time.Minute, "UpdatedAt should be recent")

	assert.NoError(t, bcrypt.CompareHashAndPassword(s.staff.user.passHash, []byte(args.Password)), "PassHash mismatch")

	events := s.staff.GetUncommittedEvents()
	require.Len(t, events, 1, "expected one event")
	assert.IsType(t, &StaffInvitationAccepted{}, events[0], "expected StaffRegistered event type")
	staffRegisteredEvent := events[0].(*StaffInvitationAccepted)
	assert.Equal(t, s.staff.user.id, staffRegisteredEvent.StaffID, "StaffID in event mismatch")
	assert.Equal(t, args.Barcode, staffRegisteredEvent.StaffBarcode, "StaffBarcode in event mismatch")
	assert.Equal(t, args.Username, staffRegisteredEvent.StaffUsername, "StaffUsername in event mismatch")
	assert.Equal(t, args.Email, staffRegisteredEvent.Email, "Email in event mismatch")
	assert.Equal(t, args.FirstName, staffRegisteredEvent.FirstName, "FirstName in event mismatch")
	assert.Equal(t, args.LastName, staffRegisteredEvent.LastName, "LastName in event mismatch")
	assert.Equal(t, args.InvitationID, staffRegisteredEvent.InvitationID, "InvitationID in event mismatch")

	return s
}

func (s *StaffAssertions) AssertByCreateInitialArgs(t *testing.T, args CreateInitialStaffArgs) *StaffAssertions {
	t.Helper()
	assert.NotEmpty(t, s.staff.user.id, "ID should not be empty")
	assert.Equal(t, args.Barcode, s.staff.user.barcode, "Barcode mismatch")
	assert.Equal(t, args.Username, s.staff.user.username, "Username mismatch")
	assert.Equal(t, args.FirstName, s.staff.user.firstName, "FirstName mismatch")
	assert.Equal(t, args.LastName, s.staff.user.lastName, "LastName mismatch")
	assert.Equal(t, args.Email, s.staff.user.email, "Email mismatch")
	assert.Equal(t, role.Staff, s.staff.user.role, "Role mismatch")
	assert.WithinDuration(t, time.Now(), s.staff.user.createdAt, time.Minute, "CreatedAt should be recent")
	assert.WithinDuration(t, time.Now(), s.staff.user.updatedAt, time.Minute, "UpdatedAt should be recent")

	assert.NoError(t, bcrypt.CompareHashAndPassword(s.staff.user.passHash, []byte(args.Password)), "PassHash mismatch")

	events := s.staff.GetUncommittedEvents()
	require.Len(t, events, 1, "expected one event")
	assert.IsType(t, &InitialStaffCreated{}, events[0], "expected InitialStaffCreated event type")
	initialStaffCreatedEvent := events[0].(*InitialStaffCreated)
	assert.Equal(t, s.staff.user.id, initialStaffCreatedEvent.StaffID, "StaffID in event mismatch")
	assert.Equal(t, args.Barcode, initialStaffCreatedEvent.StaffBarcode, "StaffBarcode in event mismatch")
	assert.Equal(t, args.Username, initialStaffCreatedEvent.StaffUsername, "StaffUsername in event mismatch")
	assert.Equal(t, args.Email, initialStaffCreatedEvent.Email, "Email in event mismatch")
	assert.Equal(t, args.FirstName, initialStaffCreatedEvent.FirstName, "FirstName in event mismatch")
	assert.Equal(t, args.LastName, initialStaffCreatedEvent.LastName, "LastName in event mismatch")

	return s
}

func (s *StaffAssertions) AssertIDNotEmpty(t *testing.T) *StaffAssertions {
	t.Helper()
	assert.NotEmpty(t, s.staff.user.id, "ID should not be empty")
	return s
}

func (s *StaffAssertions) AssertBarcode(t *testing.T, expected Barcode) *StaffAssertions {
	t.Helper()
	assert.Equal(t, expected, s.staff.user.barcode, "Barcode mismatch")
	return s
}

func (s *StaffAssertions) AssertUsername(t *testing.T, expected string) *StaffAssertions {
	t.Helper()
	assert.Equal(t, expected, s.staff.user.username, "Username mismatch")
	return s
}

func (s *StaffAssertions) AssertFirstName(t *testing.T, expected string) *StaffAssertions {
	t.Helper()
	assert.Equal(t, expected, s.staff.user.firstName, "FirstName mismatch")
	return s
}

func (s *StaffAssertions) AssertLastName(t *testing.T, expected string) *StaffAssertions {
	t.Helper()
	assert.Equal(t, expected, s.staff.user.lastName, "LastName mismatch")
	return s
}

func (s *StaffAssertions) AssertAvatarURL(t *testing.T, expected string) *StaffAssertions {
	t.Helper()
	assert.Equal(t, expected, s.staff.user.avatarURL, "AvatarURL mismatch")
	return s
}

func (s *StaffAssertions) AssertEmail(t *testing.T, expected string) *StaffAssertions {
	t.Helper()
	assert.Equal(t, expected, s.staff.user.email, "Email mismatch")
	return s
}

func (s *StaffAssertions) AssertRole(t *testing.T, expected role.Global) *StaffAssertions {
	t.Helper()
	assert.Equal(t, expected, s.staff.user.role, "Role mismatch")
	return s
}

func (s *StaffAssertions) AssertPassword(t *testing.T, expected string) *StaffAssertions {
	t.Helper()
	err := bcrypt.CompareHashAndPassword(s.staff.user.passHash, []byte(expected))
	assert.NoError(t, err, "PassHash mismatch")
	return s
}

func (s *StaffAssertions) AssertPassHash(t *testing.T, expected []byte) *StaffAssertions {
	assert.Equal(t, expected, s.staff.user.passHash, "PassHash mismatch")
	return s
}

type StudentAssertions struct {
	student *Student
}

func NewStudentAssertions(s *Student) *StudentAssertions {
	return &StudentAssertions{student: s}
}

func (s *StudentAssertions) AssertByRegistrationArgs(t *testing.T, args RegisterStudentArgs) *StudentAssertions {
	t.Helper()
	assert.Equal(t, args.Barcode, s.student.user.barcode, "Barcode mismatch")
	assert.Equal(t, args.Username, s.student.user.username, "Username mismatch")
	assert.Equal(t, args.FirstName, s.student.user.firstName, "FirstName mismatch")
	assert.Equal(t, args.LastName, s.student.user.lastName, "LastName mismatch")
	assert.Equal(t, args.AvatarURL, s.student.user.avatarURL, "AvatarURL mismatch")
	assert.Equal(t, args.Email, s.student.user.email, "Email mismatch")
	assert.Equal(t, args.GroupID, s.student.groupID, "GroupID mismatch")
	assert.Equal(t, role.Student, s.student.user.role, "Role mismatch")
	assert.NoError(t, bcrypt.CompareHashAndPassword(s.student.user.passHash, []byte(args.Password)), "PassHash mismatch")

	events := s.student.GetUncommittedEvents()
	require.Len(t, events, 1, "expected one event")
	assert.IsType(t, &StudentRegistered{}, events[0], "expected StudentRegistered event type")
	studentRegisteredEvent := events[0].(*StudentRegistered)
	assert.Equal(t, s.student.user.id, studentRegisteredEvent.StudentID, "StudentID in event mismatch")
	assert.Equal(t, args.Barcode, studentRegisteredEvent.StudentBarcode, "StudentBarcode in event mismatch")
	assert.Equal(t, args.Username, studentRegisteredEvent.StudentUsername, "StudentUsername in event mismatch")
	assert.Equal(t, args.RegistrationID, studentRegisteredEvent.RegistrationID, "RegistrationID in event mismatch")
	assert.Equal(t, args.Email, studentRegisteredEvent.Email, "Email in event mismatch")
	assert.Equal(t, args.FirstName, studentRegisteredEvent.FirstName, "FirstName in event mismatch")
	assert.Equal(t, args.LastName, studentRegisteredEvent.LastName, "LastName in event mismatch")
	assert.Equal(t, args.GroupID, studentRegisteredEvent.GroupID, "GroupID in event mismatch")

	return s
}

func (s *StudentAssertions) AssertIDNotEmpty(t *testing.T) *StudentAssertions {
	t.Helper()
	assert.NotEmpty(t, s.student.user.id, "ID should not be empty")
	return s
}

func (s *StudentAssertions) AssertBarcode(t *testing.T, expected Barcode) *StudentAssertions {
	t.Helper()
	assert.Equal(t, expected, s.student.user.barcode, "Barcode mismatch")
	return s
}

func (s *StudentAssertions) AssertUsername(t *testing.T, expected string) *StudentAssertions {
	t.Helper()
	assert.Equal(t, expected, s.student.user.username, "Username mismatch")
	return s
}

func (s *StudentAssertions) AssertFirstName(t *testing.T, expected string) *StudentAssertions {
	t.Helper()
	assert.Equal(t, expected, s.student.user.firstName, "FirstName mismatch")
	return s
}

func (s *StudentAssertions) AssertLastName(t *testing.T, expected string) *StudentAssertions {
	t.Helper()
	assert.Equal(t, expected, s.student.user.lastName, "LastName mismatch")
	return s
}

func (s *StudentAssertions) AssertAvatarURL(t *testing.T, expected string) *StudentAssertions {
	t.Helper()
	assert.Equal(t, expected, s.student.user.avatarURL, "AvatarURL mismatch")
	return s
}

func (s *StudentAssertions) AssertEmail(t *testing.T, expected string) *StudentAssertions {
	t.Helper()
	assert.Equal(t, expected, s.student.user.email, "Email mismatch")
	return s
}

func (s *StudentAssertions) AssertGroupID(t *testing.T, expected group.ID) *StudentAssertions {
	t.Helper()
	assert.Equal(t, expected, s.student.groupID, "GroupID mismatch")
	return s
}

func (s *StudentAssertions) AssertRole(t *testing.T, expected role.Global) *StudentAssertions {
	t.Helper()
	assert.Equal(t, expected, s.student.user.role, "Role mismatch")
	return s
}

func (s *StudentAssertions) AssertPassword(t *testing.T, expected string) *StudentAssertions {
	t.Helper()
	err := bcrypt.CompareHashAndPassword(s.student.user.passHash, []byte(expected))
	assert.NoError(t, err, "PassHash mismatch")
	return s
}

func (s *StudentAssertions) AssertPassHash(t *testing.T, expected []byte) *StudentAssertions {
	t.Helper()
	assert.Equal(t, expected, s.student.user.passHash, "PassHash mismatch")
	return s
}

type StudentRegistrationAssertions struct {
	t     *testing.T
	event *StudentRegistered
}

func NewStudentRegistrationAssertions(t *testing.T, event *StudentRegistered) *StudentRegistrationAssertions {
	return &StudentRegistrationAssertions{
		t:     t,
		event: event,
	}
}

func (s *StudentRegistrationAssertions) AssertStudentID(expected ID) *StudentRegistrationAssertions {
	s.t.Helper()
	assert.Equal(s.t, expected, s.event.StudentID, "StudentID mismatch")
	return s
}

func (s *StudentRegistrationAssertions) AssertStudentBarcode(expected Barcode) *StudentRegistrationAssertions {
	s.t.Helper()
	assert.Equal(s.t, expected, s.event.StudentBarcode, "StudentBarcode mismatch")
	return s
}

func (s *StudentRegistrationAssertions) AssertStudentUsername(expected string) *StudentRegistrationAssertions {
	s.t.Helper()
	assert.Equal(s.t, expected, s.event.StudentUsername, "StudentUsername mismatch")
	return s
}

func (s *StudentRegistrationAssertions) AssertRegistrationID(expected registration.ID) *StudentRegistrationAssertions {
	s.t.Helper()
	assert.Equal(s.t, expected, s.event.RegistrationID, "RegistrationID mismatch")
	return s
}

func (s *StudentRegistrationAssertions) AssertEmail(expected string) *StudentRegistrationAssertions {
	s.t.Helper()
	assert.Equal(s.t, expected, s.event.Email, "Email mismatch")
	return s
}

func (s *StudentRegistrationAssertions) AssertFirstName(expected string) *StudentRegistrationAssertions {
	s.t.Helper()
	assert.Equal(s.t, expected, s.event.FirstName, "FirstName mismatch")
	return s
}

func (s *StudentRegistrationAssertions) AssertLastName(expected string) *StudentRegistrationAssertions {
	s.t.Helper()
	assert.Equal(s.t, expected, s.event.LastName, "LastName mismatch")
	return s
}

func (s *StudentRegistrationAssertions) AssertGroupID(expected group.ID) *StudentRegistrationAssertions {
	s.t.Helper()
	assert.Equal(s.t, expected, s.event.GroupID, "GroupID mismatch")
	return s
}
