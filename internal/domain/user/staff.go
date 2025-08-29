package user

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
	"github.com/ARUMANDESU/ucms/pkg/validationx"
)

type Staff struct {
	event.Recorder
	user User
}

type RegisterStaffArgs struct {
	Barcode        Barcode         `json:"barcode"`
	Username       string          `json:"username"`
	RegistrationID registration.ID `json:"registration_id"`
	FirstName      string          `json:"first_name"`
	LastName       string          `json:"last_name"`
	AvatarURL      string          `json:"avatar_url"`
	Email          string          `json:"email"`
	Password       string          `json:"password"`
}

func RegisterStaff(p RegisterStaffArgs) (*Staff, error) {
	err := validation.ValidateStruct(&p,
		validation.Field(&p.Barcode, validation.Required),
		validation.Field(&p.Username, validation.Required, validationx.IsUsername),
		validation.Field(&p.RegistrationID, validationx.Required),
		validation.Field(&p.Email, validation.Required, is.EmailFormat),
		validation.Field(&p.FirstName, validation.Required, validation.Length(MinFirstNameLen, MaxFirstNameLen)),
		validation.Field(&p.LastName, validation.Required, validation.Length(MinLastNameLen, MaxLastNameLen)),
		validation.Field(&p.Password, validationx.PasswordRules...),
		validation.Field(&p.AvatarURL, validation.Length(0, 1000)),
	)
	if err != nil {
		return nil, err
	}

	passhash, err := NewPasswordHash(p.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	staff := &Staff{
		user: User{
			id:        NewID(),
			barcode:   p.Barcode,
			username:  p.Username,
			firstName: p.FirstName,
			lastName:  p.LastName,
			avatarURL: p.AvatarURL,
			role:      role.Staff,
			email:     p.Email,
			passHash:  passhash,
			createdAt: now,
			updatedAt: now,
		},
	}

	staff.AddEvent(&StaffRegistered{
		Header:         event.NewEventHeader(),
		StaffID:        staff.user.id,
		StaffBarcode:   p.Barcode,
		StaffUsername:  p.Username,
		RegistrationID: p.RegistrationID,
		FirstName:      p.FirstName,
		LastName:       p.LastName,
		Email:          p.Email,
	})

	return staff, nil
}

type RehydrateStaffArgs struct {
	RehydrateUserArgs
}

func RehydrateStaff(p RehydrateStaffArgs) *Staff {
	return &Staff{
		user: *RehydrateUser(p.RehydrateUserArgs),
	}
}

func (s *Staff) User() *User {
	if s == nil {
		return nil
	}
	return &s.user
}
