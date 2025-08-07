package user

import (
	"time"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
)

type Staff struct {
	event.Recorder
	user User
}

type RegisterStaffArgs struct {
	ID        ID
	FirstName string
	LastName  string
	AvatarURL string
	Email     string
	PassHash  []byte
}

func RegisterStaff(p RegisterStaffArgs) (*Staff, error) {
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
