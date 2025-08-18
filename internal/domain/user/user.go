package user

import (
	"errors"
	"fmt"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"golang.org/x/crypto/bcrypt"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
)

const PasswordCostFactor = 12 // Future-proofing; default is 10 in 2025.08.18

const (
	MaxFirstNameLen = 100
	MinFirstNameLen = 2
	MaxLastNameLen  = 100
	MinLastNameLen  = 2
	MaxBarcodeLen   = 100
	MinBarcodeLen   = 6
	MaxAvatarURLLen = 1000
)

type ID string

func (id ID) String() string {
	if id == "" {
		return ""
	}
	return string(id)
}

type User struct {
	event.Recorder
	id        ID
	firstName string
	lastName  string
	avatarURL string
	role      role.Global
	email     string
	passHash  []byte
	createdAt time.Time
	updatedAt time.Time
}

type RehydrateUserArgs struct {
	ID        ID
	FirstName string
	LastName  string
	Role      role.Global
	AvatarURL string
	Email     string
	PassHash  []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}

func RehydrateUser(p RehydrateUserArgs) *User {
	return &User{
		id:        p.ID,
		firstName: p.FirstName,
		lastName:  p.LastName,
		role:      p.Role,
		avatarURL: p.AvatarURL,
		email:     p.Email,
		passHash:  p.PassHash,
		createdAt: p.CreatedAt,
		updatedAt: p.UpdatedAt,
	}
}

func (u *User) SetFirstName(firstName string) error {
	if u == nil {
		return errors.New("user is nil")
	}
	err := validation.Validate(firstName, validation.Required, validation.Length(MinFirstNameLen, MaxFirstNameLen))
	if err != nil {
		return err
	}

	u.firstName = firstName
	u.updatedAt = time.Now().UTC()
	return nil
}

func (u *User) SetLastName(lastName string) error {
	if u == nil {
		return errors.New("user is nil")
	}
	err := validation.Validate(lastName, validation.Required, validation.Length(MinLastNameLen, MaxLastNameLen))
	if err != nil {
		return err
	}

	u.lastName = lastName
	u.updatedAt = time.Now().UTC()
	return nil
}

func (u *User) SetAvatarURL(avatarURL string) error {
	if u == nil {
		return errors.New("user is nil")
	}
	err := validation.Validate(avatarURL, validation.Length(1, MaxAvatarURLLen))
	if err != nil {
		return err
	}

	u.avatarURL = avatarURL
	u.updatedAt = time.Now().UTC()
	return nil
}

func (u *User) ComparePassword(password string) error {
	return bcrypt.CompareHashAndPassword(u.passHash, []byte(password))
}

func (u *User) ID() ID {
	if u == nil {
		return ""
	}

	return u.id
}

func (u *User) FirstName() string {
	if u == nil {
		return ""
	}

	return u.firstName
}

func (u *User) LastName() string {
	if u == nil {
		return ""
	}

	return u.lastName
}

func (u *User) Role() role.Global {
	if u == nil {
		return ""
	}

	return u.role
}

func (u *User) AvatarUrl() string {
	if u == nil {
		return ""
	}

	return u.avatarURL
}

func (u *User) Email() string {
	if u == nil {
		return ""
	}

	return u.email
}

func (u *User) PassHash() []byte {
	if u == nil {
		return nil
	}

	return u.passHash
}

func (u *User) CreatedAt() time.Time {
	if u == nil {
		return time.Time{}
	}

	return u.createdAt
}

func (u *User) UpdatedAt() time.Time {
	if u == nil {
		return time.Time{}
	}

	return u.updatedAt
}

func NewPasswordHash(password string) ([]byte, error) {
	passhash, err := bcrypt.GenerateFromPassword([]byte(password), PasswordCostFactor)
	if err != nil {
		return nil, fmt.Errorf("failed to generate password hash from password: %w", err)
	}
	return passhash, nil
}
