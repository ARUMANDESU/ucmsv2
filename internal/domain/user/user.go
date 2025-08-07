package user

import (
	"errors"
	"time"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
)

const (
	MaxFirstNameLen = 100
	MinFirstNameLen = 2
	MaxLastNameLen  = 100
	MinLastNameLen  = 2
)

type ID string

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
	if len([]rune(firstName)) > MaxFirstNameLen {
		return errors.New("first name is too long")
	}
	if len([]rune(firstName)) < MinFirstNameLen {
		return errors.New("first name is too short")
	}

	u.firstName = firstName
	u.updatedAt = time.Now().UTC()
	return nil
}

func (u *User) SetLastName(lastName string) error {
	if u == nil {
		return errors.New("user is nil")
	}
	if len([]rune(lastName)) > MaxLastNameLen {
		return errors.New("last name is too long")
	}
	if len([]rune(lastName)) < MinLastNameLen {
		return errors.New("last name is too short")
	}

	u.lastName = lastName
	u.updatedAt = time.Now().UTC()
	return nil
}

func (u *User) SetAvatarURL(avatarURL string) error {
	if u == nil {
		return errors.New("user is nil")
	}

	u.avatarURL = avatarURL
	u.updatedAt = time.Now().UTC()
	return nil
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
