package user

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
)

type Student struct {
	event.Recorder
	user    User
	groupID uuid.UUID
}

type RegisterStudentArgs struct {
	ID        ID        `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	AvatarURL string    `json:"avatar_url"`
	Email     string    `json:"email"`
	PassHash  []byte    `json:"pass_hash"`
	GroupID   uuid.UUID `json:"group_id"`
}

func RegisterStudent(p RegisterStudentArgs) (*Student, error) {
	err := validation.ValidateStruct(&p,
		validation.Field(&p.ID, validation.Required),
		validation.Field(&p.Email, validation.Required),
		validation.Field(&p.FirstName, validation.Required, validation.Length(MinFirstNameLen, MaxFirstNameLen), is.Alphanumeric),
		validation.Field(&p.LastName, validation.Required, validation.Length(MinLastNameLen, MaxLastNameLen), is.Alphanumeric),
		validation.Field(&p.PassHash, validation.Required),
		validation.Field(&p.GroupID, validation.Required, validation.By(errorx.ValidateGroupID)),
		validation.Field(&p.AvatarURL, validation.Length(0, 1000)),
	)
	if err != nil {
		return nil, err
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
	err := validation.Validate(groupID, validation.Required, validation.By(errorx.ValidateGroupID))
	if err != nil {
		return err
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
