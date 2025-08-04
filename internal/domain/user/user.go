package user

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
)

const (
	PasswordCostFactor = 12 // Future-proofing; default is 10 in 2025.07.30
	MaxFirstNameLen    = 100
	MinFirstNameLen    = 2
	MaxLastNameLen     = 100
	MinLastNameLen     = 2
)

type ID string

type User struct {
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

type RegisterUserArgs struct {
	ID        ID
	FirstName string
	LastName  string
	AvatarURL string
	Email     string
	Password  string
}

func RegisterUser(p RegisterUserArgs) (*User, error) {
	if p.ID == "" {
		return nil, errors.New("id is required")
	}
	if p.Email == "" {
		return nil, errors.New("email is required")
	}
	if len(p.Password) == 0 {
		return nil, errors.New("password hash is required")
	}
	if p.FirstName == "" {
		return nil, errors.New("first name is required")
	}
	if len([]rune(p.FirstName)) > MaxFirstNameLen {
		return nil, errors.New("first name is too long")
	}
	if len([]rune(p.FirstName)) < MinFirstNameLen {
		return nil, errors.New("first name is too short")
	}
	if p.LastName == "" {
		return nil, errors.New("last name is required")
	}
	if len([]rune(p.LastName)) > MaxLastNameLen {
		return nil, errors.New("last name is too long")
	}
	if len([]rune(p.LastName)) < MinLastNameLen {
		return nil, errors.New("last name is too short")
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(p.Password), PasswordCostFactor)
	if err != nil {
		return nil, fmt.Errorf("failed to generate password hash: %w", err)
	}

	now := time.Now().UTC()

	return &User{
		id:        p.ID,
		firstName: p.FirstName,
		lastName:  p.LastName,
		avatarURL: p.AvatarURL,
		role:      role.StudentRole,
		email:     p.Email,
		passHash:  passHash,
		createdAt: now,
		updatedAt: now,
	}, nil
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

func (u *User) SetAvatarURL(avatarURL string) {
	if u == nil {
		return
	}

	u.avatarURL = avatarURL
	u.updatedAt = time.Now().UTC()
}

func (u *User) SetEmail(email string) error {
	if u == nil {
		return errors.New("user is nil")
	}
	if email == "" {
		return errors.New("email is required")
	}

	u.email = email
	u.updatedAt = time.Now().UTC()
	return nil
}

func (u *User) SetPassHash(pass string) error {
	if u == nil {
		return errors.New("user is nil")
	}
	if len(pass) == 0 {
		return errors.New("password is required")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pass), PasswordCostFactor)
	if err != nil {
		return fmt.Errorf("failed to generate password hash: %w", err)
	}

	u.passHash = hash
	u.updatedAt = time.Now().UTC()
	return nil
}
