package registration

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/publicsuffix"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/pkg/env"
	"github.com/ARUMANDESU/ucms/pkg/randcode"
)

var emailRx = regexp.MustCompile(
	`^[a-zA-Z0-9._%+\-]+@` + // local part
		`(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.)+` + // labels
		`[A-Za-z]{2,63}$`) // TLD

const (
	PasswordCostFactor     = 12 // Future-proofing; default is 10 in 2025.07.30
	VerificationCodeLength = 6

	ResendTimeout               = 1 * time.Minute
	ExpiresAt                   = 10 * time.Minute
	MaxEmailLength              = 254 // Maximum length for email addresses as per RFC 5321
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
	if email == "" {
		return nil, ErrEmptyEmail
	}
	if len(email) > MaxEmailLength {
		return nil, ErrEmailExceedsMaxLength
	}
	if !emailRx.MatchString(email) {
		return nil, ErrInvalidEmailFormat
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEmailParseFailed, err)
	}

	code, err := generateCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification code: %w", err)
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
	if r.status != StatusPending {
		return fmt.Errorf("%w: %w", ErrInvalidStatus, errors.New("can only verify pending registrations"))
	}

	if time.Now().After(r.codeExpiresAt) {
		r.status = StatusExpired
		return ErrCodeExpired
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
			return ErrPersistentTooManyAttempts
		}
		return ErrPersistentVerificationCodeMismatch
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

type StudentArgs struct {
	VerificationCode string
	Barcode          string
	FirstName        string
	LastName         string
	Password         string
	GroupID          uuid.UUID
}

func (r *Registration) CompleteStudentRegistration(args StudentArgs) error {
	if r.verificationCode != args.VerificationCode {
		return ErrInvalidVerificationCode
	}
	if !r.IsStatus(StatusVerified) {
		if err := r.VerifyCode(args.VerificationCode); err != nil {
			return fmt.Errorf("failed to verify code: %w", err)
		}
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(args.Password), PasswordCostFactor)
	if err != nil {
		return fmt.Errorf("failed to generate password hash: %w", err)
	}

	r.status = StatusCompleted
	r.AddEvent(&StudentRegistrationCompleted{
		Header:         event.NewEventHeader(),
		RegistrationID: r.id,
		Barcode:        args.Barcode,
		Email:          r.email,
		FirstName:      args.FirstName,
		LastName:       args.LastName,
		PassHash:       passHash,
		GroupID:        args.GroupID,
	})

	return nil
}

type StaffArgs struct {
	VerificationCode string
	Barcode          string
	FirstName        string
	LastName         string
	Password         string
}

func (r *Registration) CompleteStaffRegistration(args StaffArgs) error {
	if r.verificationCode != args.VerificationCode {
		return ErrInvalidVerificationCode
	}
	if !r.IsStatus(StatusVerified) {
		if err := r.VerifyCode(args.VerificationCode); err != nil {
			return fmt.Errorf("failed to verify code: %w", err)
		}
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(args.Password), PasswordCostFactor)
	if err != nil {
		return fmt.Errorf("failed to generate password hash: %w", err)
	}

	r.status = StatusCompleted
	r.AddEvent(&StaffRegistrationCompleted{
		Header:         event.NewEventHeader(),
		RegistrationID: r.id,
		Barcode:        args.Barcode,
		Email:          r.email,
		FirstName:      args.FirstName,
		LastName:       args.LastName,
		PassHash:       passHash,
	})

	return nil
}

func (r *Registration) ResendCode() error {
	if !r.resendTimeout.IsZero() && !time.Now().After(r.resendTimeout) {
		return fmt.Errorf("%w: time left until next resend: %s", ErrWaitUntilResend, time.Until(r.resendTimeout).String())
	}

	if r.IsCompleted() {
		return ErrRegistrationCompleted
	}

	code, err := generateCode()
	if err != nil {
		return fmt.Errorf("failed to generate new verification code: %w", err)
	}
	fmt.Println("Generated new verification code:", code)

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
	code, err := randcode.GenerateAlphaNumericCode(VerificationCodeLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate new verification code: %w", err)
	}

	return code, nil
}

func hasRealTLD(addr string) bool {
	parsed, err := mail.ParseAddress(addr)
	if err != nil {
		return false
	}

	at := strings.LastIndexByte(parsed.Address, '@')
	domain := parsed.Address[at+1:]

	// Ask PSL what the public suffix is and whether it’s ICANN‑managed
	suffix, icann := publicsuffix.PublicSuffix(domain)

	// If the suffix is the entire domain, there's no registrable part,
	// so "localhost", "internal", etc. will fail here.
	return icann && suffix != domain
}
