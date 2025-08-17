package user

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
)

type Staff struct {
	event.Recorder
	user User
}

type RegisterStaffArgs struct {
	ID        ID     `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
	PassHash  []byte `json:"pass_hash"`
}

func RegisterStaff(p RegisterStaffArgs) (*Staff, error) {
	err := validation.ValidateStruct(&p,
		validation.Field(&p.ID, validation.Required),
		validation.Field(&p.Email, validation.Required),
		validation.Field(&p.FirstName, validation.Required, validation.Length(MinFirstNameLen, MaxFirstNameLen), is.Alphanumeric),
		validation.Field(&p.LastName, validation.Required, validation.Length(MinLastNameLen, MaxLastNameLen), is.Alphanumeric),
		validation.Field(&p.PassHash, validation.Required),
		validation.Field(&p.AvatarURL, validation.Length(0, 1000)),
	)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	staff := &Staff{
		user: User{
			id:        p.ID,
			firstName: p.FirstName,
			lastName:  p.LastName,
			avatarURL: p.AvatarURL,
			role:      role.Staff,
			email:     p.Email,
			passHash:  p.PassHash,
			createdAt: now,
			updatedAt: now,
		},
	}

	staff.AddEvent(&StaffRegistered{
		Header:    event.NewEventHeader(),
		StaffID:   p.ID,
		FirstName: p.FirstName,
		LastName:  p.LastName,
		Email:     p.Email,
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
