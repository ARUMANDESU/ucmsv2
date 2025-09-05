package registration

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
	"github.com/google/uuid"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/event"
	"gitlab.com/ucmsv2/ucms-backend/pkg/env"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/randcode"
)

const (
	VerificationCodeLength = 6

	ResendTimeout               = 1 * time.Minute
	ExpiresAt                   = 10 * time.Minute
	MaxVerificationCodeAttempts = 3
)

type Status string

func (s Status) String() string {
	return string(s)
}

const (
	StatusPending   Status = "pending"
	StatusExpired   Status = "expired"
	StatusVerified  Status = "verified"
	StatusCompleted Status = "completed"
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

type Registration struct {
	event.Recorder
	id               ID
	email            string
	status           Status
	verificationCode string
	codeAttempts     int8
	resendTimeout    time.Time
	codeExpiresAt    time.Time
	createdAt        time.Time
	updatedAt        time.Time
}

func NewRegistration(email string, mode env.Mode) (*Registration, error) {
	const op = "registration.NewRegistration"
	err := validation.Validate(&email, validation.Required, is.Email)
	if err != nil {
		return nil, errorx.Wrap(err, op)
	}

	code, err := generateCode()
	if err != nil {
		return nil, errorx.Wrap(err, op)
	}
	now := time.Now().UTC()

	reg := &Registration{
		id:               NewID(),
		email:            email,
		status:           StatusPending,
		verificationCode: code,
		resendTimeout:    now.Add(ResendTimeout),
		codeExpiresAt:    now.Add(ExpiresAt),
		codeAttempts:     0,
		createdAt:        now,
		updatedAt:        now,
	}

	reg.AddEvent(&RegistrationStarted{
		Header:           event.NewEventHeader(),
		RegistrationID:   reg.id,
		Email:            email,
		VerificationCode: code,
	})

	return reg, nil
}

type RehydrateArgs struct {
	ID               ID
	Email            string
	Status           Status
	VerificationCode string
	CodeAttempts     int8
	CodeExpiresAt    time.Time
	ResendTimeout    time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func Rehydrate(args RehydrateArgs) *Registration {
	return &Registration{
		id:               args.ID,
		email:            args.Email,
		status:           args.Status,
		verificationCode: args.VerificationCode,
		codeAttempts:     args.CodeAttempts,
		codeExpiresAt:    args.CodeExpiresAt,
		resendTimeout:    args.ResendTimeout,
		createdAt:        args.CreatedAt,
		updatedAt:        args.UpdatedAt,
	}
}

func (r *Registration) VerifyCode(code string) error {
	const op = "registration.Registration.VerifyCode"
	if r.status != StatusPending {
		return errorx.Wrap(ErrInvalidStatus, op)
	}

	if time.Now().After(r.codeExpiresAt) {
		r.status = StatusExpired
		return errorx.Wrap(ErrCodeExpired, op)
	}

	if r.verificationCode != code {
		r.codeAttempts++
		if r.codeAttempts >= MaxVerificationCodeAttempts {
			r.status = StatusExpired
			r.AddEvent(&RegistrationFailed{
				Header:         event.NewEventHeader(),
				RegistrationID: r.id,
				Reason:         "too many failed attempts",
			})
			return errorx.Wrap(ErrPersistentTooManyAttempts, op)
		}
		return errorx.Wrap(ErrPersistentVerificationCodeMismatch, op)
	}

	r.updatedAt = time.Now().UTC()
	r.status = StatusVerified
	r.AddEvent(&EmailVerified{
		Header:         event.NewEventHeader(),
		RegistrationID: r.id,
		Email:          r.email,
	})

	return nil
}

func (r *Registration) CheckCode(code string) error {
	const op = "registration.Registration.CheckCode"
	if r.status == StatusCompleted {
		return errorx.Wrap(ErrRegistrationCompleted, op)
	}
	if r.status != StatusVerified {
		return errorx.Wrap(ErrVerifyFirst, op)
	}

	if time.Now().After(r.codeExpiresAt) {
		return errorx.Wrap(ErrCodeExpired, op)
	}

	if r.verificationCode != code {
		return errorx.Wrap(ErrInvalidVerificationCode, op)
	}

	return nil
}

func (r *Registration) ResendCode() error {
	const op = "registration.Registration.ResendCode"
	if !r.resendTimeout.IsZero() && !time.Now().After(r.resendTimeout) {
		return errorx.Wrap(ErrWaitUntilResend, op)
	}

	if r.IsCompleted() {
		return errorx.Wrap(ErrRegistrationCompleted, op)
	}

	code, err := generateCode()
	if err != nil {
		return errorx.Wrap(err, op)
	}

	r.verificationCode = code
	r.codeExpiresAt = time.Now().UTC().Add(10 * time.Minute)
	r.resendTimeout = time.Now().UTC().Add(ResendTimeout)
	r.codeAttempts = 0
	r.updatedAt = time.Now().UTC()
	r.status = StatusPending

	r.AddEvent(&VerificationCodeResent{
		Header:           event.NewEventHeader(),
		RegistrationID:   r.id,
		Email:            r.email,
		VerificationCode: code,
	})

	return nil
}

func (r *Registration) Complete() error {
	const op = "registration.Registration.Complete"
	if r == nil {
		return errorx.Wrap(errors.New("registration is nil"), op)
	}
	if r.status != StatusVerified && r.status != StatusCompleted {
		return errorx.Wrap(ErrInvalidStatus, op)
	}

	r.status = StatusCompleted
	r.updatedAt = time.Now().UTC()
	return nil
}

func (r *Registration) IsStatus(s Status) bool {
	if r == nil {
		return false
	}

	return r.status == s
}

func (r *Registration) IsCompleted() bool {
	return r.IsStatus(StatusCompleted)
}

func (r *Registration) ID() ID {
	if r == nil {
		return ID{}
	}

	return r.id
}

func (r *Registration) Email() string {
	if r == nil {
		return ""
	}

	return r.email
}

func (r *Registration) Status() Status {
	if r == nil {
		return ""
	}

	return r.status
}

func (r *Registration) VerificationCode() string {
	if r == nil {
		return ""
	}

	return r.verificationCode
}

func (r *Registration) CodeAttempts() int8 {
	if r == nil {
		return 0
	}

	return r.codeAttempts
}

func (r *Registration) CodeExpiresAt() time.Time {
	if r == nil {
		return time.Time{}
	}

	return r.codeExpiresAt
}

func (r *Registration) ResendTimeout() time.Time {
	if r == nil {
		return time.Time{}
	}

	return r.resendTimeout
}

func (r *Registration) CreatedAt() time.Time {
	if r == nil {
		return time.Time{}
	}

	return r.createdAt
}

func (r *Registration) UpdatedAt() time.Time {
	if r == nil {
		return time.Time{}
	}

	return r.updatedAt
}

func generateCode() (string, error) {
	const op = "registration.generateCode"
	code, err := randcode.GenerateAlphaNumericCode(VerificationCodeLength)
	if err != nil {
		return "", errorx.Wrap(err, op)
	}

	return code, nil
}
