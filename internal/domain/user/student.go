package user

import (
	"time"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/group"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/validationx"
)

type Student struct {
	event.Recorder
	user    User
	groupID group.ID
}

type RegisterStudentArgs struct {
	Barcode        Barcode         `json:"barcode"`
	Username       string          `json:"username"`
	RegistrationID registration.ID `json:"registration_id"`
	FirstName      string          `json:"first_name"`
	LastName       string          `json:"last_name"`
	AvatarURL      string          `json:"avatar_url"`
	Email          string          `json:"email"`
	Password       string          `json:"password"`
	GroupID        group.ID        `json:"group_id"`
}

func RegisterStudent(p RegisterStudentArgs) (*Student, error) {
	const op = "user.RegisterStudent"
	err := validation.ValidateStruct(&p,
		validation.Field(&p.Username, validation.Required, validationx.IsUsername),
		validation.Field(&p.Barcode, validation.Required, validation.Length(6, 20), is.Alphanumeric),
		validation.Field(&p.RegistrationID, validationx.Required),
		validation.Field(&p.Email, validation.Required, is.EmailFormat),
		validation.Field(&p.FirstName, validation.Required, validation.Length(MinFirstNameLen, MaxFirstNameLen)),
		validation.Field(&p.LastName, validation.Required, validation.Length(MinLastNameLen, MaxLastNameLen)),
		validation.Field(&p.Password, validationx.PasswordRules...),
		validation.Field(&p.GroupID, validationx.Required),
		validation.Field(&p.AvatarURL, validation.Length(0, 1000)),
	)
	if err != nil {
		return nil, errorx.Wrap(err, op)
	}

	passhash, err := NewPasswordHash(p.Password)
	if err != nil {
		return nil, errorx.Wrap(err, op)
	}

	now := time.Now().UTC()

	student := &Student{
		user: User{
			id:        NewID(),
			barcode:   p.Barcode,
			username:  p.Username,
			firstName: p.FirstName,
			lastName:  p.LastName,
			avatarURL: p.AvatarURL,
			role:      role.Student,
			email:     p.Email,
			passHash:  passhash,
			createdAt: now,
			updatedAt: now,
		},
		groupID: p.GroupID,
	}

	student.AddEvent(&StudentRegistered{
		Header:          event.NewEventHeader(),
		StudentID:       student.user.id,
		StudentBarcode:  p.Barcode,
		StudentUsername: p.Username,
		RegistrationID:  p.RegistrationID,
		Email:           p.Email,
		FirstName:       p.FirstName,
		LastName:        p.LastName,
		GroupID:         p.GroupID,
	})

	return student, nil
}

type RehydrateStudentArgs struct {
	RehydrateUserArgs
	GroupID group.ID
}

func RehydrateStudent(p RehydrateStudentArgs) *Student {
	return &Student{
		user:    *RehydrateUser(p.RehydrateUserArgs),
		groupID: p.GroupID,
	}
}

func (s *Student) SetGroupID(groupID group.ID) error {
	const op = "user.Student.SetGroupID"
	err := validation.Validate(groupID, validationx.Required)
	if err != nil {
		return errorx.Wrap(err, op)
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

func (s *Student) GroupID() group.ID {
	if s == nil {
		return group.ID{}
	}

	return s.groupID
}
