package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ARUMANDESU/validation"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/event"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/valueobject/avatars"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/valueobject/roles"
	"gitlab.com/ucmsv2/ucms-backend/pkg/env"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/sanitizex"
)

const (
	PasswordCostFactor  = 12 // Future-proofing; default is 10 in 2025.08.18
	UserEventStreamName = "events_user"
)

const (
	MaxFirstNameLen   = 100
	MinFirstNameLen   = 2
	MaxLastNameLen    = 100
	MinLastNameLen    = 2
	MaxBarcodeLen     = 100
	MinBarcodeLen     = 6
	MaxAvatarS3KeyLen = 255
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
	avatar    avatars.Avatar
	role      roles.Global
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
	Role      roles.Global
	Avatar    avatars.Avatar
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
		avatar:    p.Avatar,
		email:     p.Email,
		passHash:  p.PassHash,
		createdAt: p.CreatedAt,
		updatedAt: p.UpdatedAt,
	}
}

func (u *User) SetAvatarFromS3(s3Key string) error {
	const op = "user.User.SetAvatarFromS3"
	if u == nil {
		return errorx.Wrap(errors.New("user is nil"), op)
	}
	s3Key = sanitizex.CleanSingleLine(s3Key)
	err := validation.Validate(s3Key, validation.Required, validation.Length(1, MaxAvatarS3KeyLen))
	if err != nil {
		return errorx.Wrap(err, op)
	}

	oldAvatar := u.avatar
	u.avatar = avatars.Avatar{
		Source:   avatars.SourceS3,
		S3Key:    s3Key,
		External: "",
	}
	u.updatedAt = time.Now().UTC()

	u.AddEvent(&UserAvatarUpdated{
		Header:    event.NewEventHeader(),
		UserID:    u.id,
		NewAvatar: u.avatar,
		OldAvatar: oldAvatar,
	})
	return nil
}

func (u *User) DeleteAvatar() error {
	const op = "user.User.DeleteAvatar"
	if u == nil {
		return errorx.Wrap(errors.New("user is nil"), op)
	}
	if u.avatar.IsZero() {
		return errorx.NewNotFound().WithDetails("user avatar does not exist").WithOp(op)
	}

	oldAvatar := u.avatar
	u.avatar = avatars.Avatar{
		Source:   avatars.SourceUnknown,
		S3Key:    "",
		External: "",
	}
	u.updatedAt = time.Now().UTC()

	u.AddEvent(&UserAvatarUpdated{
		Header:    event.NewEventHeader(),
		UserID:    u.id,
		NewAvatar: u.avatar,
		OldAvatar: oldAvatar,
	})
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

func (u *User) Role() roles.Global {
	if u == nil {
		return ""
	}

	return u.role
}

func (u *User) Avatar() avatars.Avatar {
	if u == nil {
		return avatars.Avatar{}
	}

	return u.avatar
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

type UserAvatarUpdated struct {
	event.Header
	event.Otel
	UserID    ID             `json:"user_id"`
	NewAvatar avatars.Avatar `json:"new_avatar"`
	OldAvatar avatars.Avatar `json:"old_avatar"`
}

func (e *UserAvatarUpdated) GetStreamName() string {
	return UserEventStreamName
}
