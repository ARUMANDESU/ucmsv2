package user

import (
	"time"

	"github.com/google/uuid"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
)

type Student struct {
	event.Recorder
	user    User
	groupID uuid.UUID
}

type RegisterStudentArgs struct {
	ID        ID
	FirstName string
	LastName  string
	AvatarURL string
	Email     string
	PassHash  []byte
	GroupID   uuid.UUID
}

func RegisterStudent(p RegisterStudentArgs) (*Student, error) {
	if p.ID == "" {
		return nil, ErrMissingID
	}
	if p.Email == "" {
		return nil, ErrMissingEmail
	}
	if len(p.PassHash) == 0 {
		return nil, ErrMissingPassHash
	}
	if p.FirstName == "" {
		return nil, ErrMissingFirstName
	}
	if len([]rune(p.FirstName)) > MaxFirstNameLen {
		return nil, ErrFirstNameTooLong
	}
	if len([]rune(p.FirstName)) < MinFirstNameLen {
		return nil, ErrFirstNameTooShort
	}
	if p.LastName == "" {
		return nil, ErrMissingLastName
	}
	if len([]rune(p.LastName)) > MaxLastNameLen {
		return nil, ErrLastNameTooLong
	}
	if len([]rune(p.LastName)) < MinLastNameLen {
		return nil, ErrLastNameTooShort
	}
	if p.GroupID == uuid.Nil {
		return nil, ErrMissingGroupID
	}

	now := time.Now().UTC()

	student := &Student{
		user: User{
			id:        p.ID,
			firstName: p.FirstName,
			lastName:  p.LastName,
			avatarURL: p.AvatarURL,
			role:      role.Student,
			email:     p.Email,
			passHash:  p.PassHash,
			createdAt: now,
			updatedAt: now,
		},
		groupID: p.GroupID,
	}

	student.AddEvent(&StudentRegistered{
		Header:    event.NewEventHeader(),
		StudentID: p.ID,
		Email:     p.Email,
		FirstName: p.FirstName,
		LastName:  p.LastName,
		GroupID:   p.GroupID,
	})

	return student, nil
}

type RehydrateStudentArgs struct {
	RehydrateUserArgs
	GroupID uuid.UUID
}

func RehydrateStudent(p RehydrateStudentArgs) *Student {
	return &Student{
		user:    *RehydrateUser(p.RehydrateUserArgs),
		groupID: p.GroupID,
	}
}

func (s *Student) SetGroupID(groupID uuid.UUID) error {
	if s == nil {
		return nil
	}
	if groupID == uuid.Nil {
		return ErrMissingGroupID
	}

	s.groupID = groupID
	return nil
}

func (s *Student) User() *User {
	if s == nil {
		return nil
	}

	return &s.user
}

func (s *Student) GroupID() uuid.UUID {
	if s == nil {
		return uuid.Nil
	}

	return s.groupID
}
