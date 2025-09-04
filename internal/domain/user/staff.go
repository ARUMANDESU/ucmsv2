package user

import (
	"time"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
	"github.com/google/uuid"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/validationx"
)

type Staff struct {
	event.Recorder
	user User
}

type AcceptStaffInvitationArgs struct {
	Barcode      Barcode   `json:"barcode"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Password     string    `json:"password"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	InvitationID uuid.UUID `json:"invitation_id"`
}

func AcceptStaffInvitation(p AcceptStaffInvitationArgs) (*Staff, error) {
	const op = "user.AcceptStaffInvitation"
	err := validation.ValidateStruct(&p,
		validation.Field(&p.Barcode, validation.Required),
		validation.Field(&p.Username, validation.Required, validationx.IsUsername),
		validation.Field(&p.Email, validation.Required, is.EmailFormat),
		validation.Field(&p.FirstName, validation.Required, validation.Length(MinFirstNameLen, MaxFirstNameLen)),
		validation.Field(&p.LastName, validation.Required, validation.Length(MinLastNameLen, MaxLastNameLen)),
		validation.Field(&p.Password, validationx.PasswordRules...),
		validation.Field(&p.InvitationID, validationx.Required, is.UUID),
	)
	if err != nil {
		return nil, errorx.Wrap(err, op)
	}

	passhash, err := NewPasswordHash(p.Password)
	if err != nil {
		return nil, errorx.Wrap(err, op)
	}

	now := time.Now().UTC()

	staff := &Staff{
		user: User{
			id:        NewID(),
			barcode:   p.Barcode,
			username:  p.Username,
			firstName: p.FirstName,
			lastName:  p.LastName,
			avatarURL: "",
			role:      role.Staff,
			email:     p.Email,
			passHash:  passhash,
			createdAt: now,
			updatedAt: now,
		},
	}

	staff.AddEvent(&StaffInvitationAccepted{
		Header:        event.NewEventHeader(),
		StaffID:       staff.user.id,
		StaffBarcode:  p.Barcode,
		StaffUsername: p.Username,
		FirstName:     p.FirstName,
		LastName:      p.LastName,
		Email:         p.Email,
		InvitationID:  p.InvitationID,
	})

	return staff, nil
}

type CreateInitialStaffArgs struct {
	Email     string  `json:"email"`
	Password  string  `json:"password"`
	Barcode   Barcode `json:"barcode"`
	Username  string  `json:"username"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
}

func CreateInitialStaff(p CreateInitialStaffArgs) (*Staff, error) {
	const op = "user.CreateInitialStaff"
	err := validation.ValidateStruct(&p,
		validation.Field(&p.Barcode, validation.Required),
		validation.Field(&p.Username, validation.Required, validationx.IsUsername),
		validation.Field(&p.Email, validation.Required, is.EmailFormat),
		validation.Field(&p.FirstName, validation.Required, validation.Length(MinFirstNameLen, MaxFirstNameLen)),
		validation.Field(&p.LastName, validation.Required, validation.Length(MinLastNameLen, MaxLastNameLen)),
		validation.Field(&p.Password, validationx.PasswordRules...),
	)
	if err != nil {
		return nil, errorx.Wrap(err, op)
	}

	passhash, err := NewPasswordHash(p.Password)
	if err != nil {
		return nil, errorx.Wrap(err, op)
	}

	now := time.Now().UTC()

	staff := &Staff{
		user: User{
			id:        NewID(),
			barcode:   p.Barcode,
			username:  p.Username,
			firstName: p.FirstName,
			lastName:  p.LastName,
			avatarURL: "",
			role:      role.Staff,
			email:     p.Email,
			passHash:  passhash,
			createdAt: now,
			updatedAt: now,
		},
	}

	staff.AddEvent(&InitialStaffCreated{
		Header:        event.NewEventHeader(),
		StaffID:       staff.user.id,
		StaffBarcode:  p.Barcode,
		StaffUsername: p.Username,
		FirstName:     p.FirstName,
		LastName:      p.LastName,
		Email:         p.Email,
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
