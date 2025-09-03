package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ARUMANDESU/validation"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
	"github.com/ARUMANDESU/ucms/pkg/env"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
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

type ID uuid.UUID

func NewID() ID {
	return ID(uuid.New())
}

func (id ID) String() string {
	return uuid.UUID(id).String()
}

func (id ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(uuid.UUID(id).String())
}

func (id *ID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	uid, err := uuid.Parse(s)
	if err != nil {
		return err
	}

	*id = ID(uid)
	return nil
}

type Barcode string

func (barcode Barcode) String() string {
	if barcode == "" {
		return ""
	}
	return string(barcode)
}

type User struct {
	event.Recorder
	id        ID
	barcode   Barcode
	username  string
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
	Barcode   Barcode
	Username  string
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
		barcode:   p.Barcode,
		username:  p.Username,
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
	const op = "user.User.SetFirstName"
	if u == nil {
		return errorx.Wrap(errors.New("user is nil"), op)
	}
	err := validation.Validate(firstName, validation.Required, validation.Length(MinFirstNameLen, MaxFirstNameLen))
	if err != nil {
		return errorx.Wrap(err, op)
	}

	u.firstName = firstName
	u.updatedAt = time.Now().UTC()
	return nil
}

func (u *User) SetLastName(lastName string) error {
	const op = "user.User.SetLastName"
	if u == nil {
		return errorx.Wrap(errors.New("user is nil"), op)
	}
	err := validation.Validate(lastName, validation.Required, validation.Length(MinLastNameLen, MaxLastNameLen))
	if err != nil {
		return errorx.Wrap(err, op)
	}

	u.lastName = lastName
	u.updatedAt = time.Now().UTC()
	return nil
}

func (u *User) SetAvatarURL(avatarURL string) error {
	const op = "user.User.SetAvatarURL"
	if u == nil {
		return errorx.Wrap(errors.New("user is nil"), op)
	}
	err := validation.Validate(avatarURL, validation.Length(1, MaxAvatarURLLen))
	if err != nil {
		return errorx.Wrap(err, op)
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
		return ID{}
	}

	return u.id
}

func (u *User) Barcode() Barcode {
	if u == nil {
		return ""
	}

	return u.barcode
}

func (u *User) Username() string {
	if u == nil {
		return ""
	}

	return u.username
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
	const op = "user.NewPasswordHash"
	costFactor := PasswordCostFactor
	if env.Current() == env.Test {
		costFactor = bcrypt.MinCost
	}
	passhash, err := bcrypt.GenerateFromPassword([]byte(password), costFactor)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to generate password hash: %w", op, err)
	}
	return passhash, nil
}
