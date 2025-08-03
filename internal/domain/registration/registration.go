package registration

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
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
	ResendTimeout  = 1 * time.Minute
	ExpiresAt      = 10 * time.Minute
	MaxEmailLength = 254 // Maximum length for email addresses as per RFC 5321
)

type Status string

func (s Status) String() string {
	return string(s)
}

const (
	StatusPending   Status = "pending"
	StatusExpired   Status = "expired"
	StatusCompleted Status = "completed"
)

type ID uuid.UUID

func NewID() ID {
	return ID(uuid.New())
}

func (id ID) String() string {
	return uuid.UUID(id).String()
}

type Registration struct {
	event.Recorder
	id               ID
	email            string
	status           Status
	verificationCode string
	codeAttempts     int
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

func (r *Registration) VerifyCode(code string) error {
	if r.status != StatusPending {
		return errors.New("registration is not pending")
	}

	if time.Now().After(r.codeExpiresAt) {
		r.status = StatusExpired
		return errors.New("verification code expired")
	}

	if r.verificationCode != code {
		r.codeAttempts++
		if r.codeAttempts >= 3 {
			r.status = StatusExpired
			r.AddEvent(&RegistrationFailed{
				Header:         event.NewEventHeader(),
				RegistrationID: r.id,
				Reason:         "too many failed attempts",
			})
		}
		return errors.New("invalid verification code")
	}

	r.updatedAt = time.Now().UTC()

	r.AddEvent(&EmailVerified{
		Header:         event.NewEventHeader(),
		RegistrationID: r.id,
		Email:          r.email,
	})

	return nil
}

type StudentArgs struct {
	Barcode   string
	FirstName string
	LastName  string
	PassHash  []byte
	GroupID   uuid.UUID
}

func (r *Registration) CompleteStudentRegistration(args StudentArgs) error {
	if r.status != StatusPending {
		return errors.New("registration is not pending")
	}

	r.status = StatusCompleted
	r.AddEvent(&StudentRegistrationCompleted{
		Header:         event.NewEventHeader(),
		RegistrationID: r.id,
		Barcode:        args.Barcode,
		Email:          r.email,
		FirstName:      args.FirstName,
		LastName:       args.LastName,
		PassHash:       args.PassHash,
		GroupID:        args.GroupID,
	})

	return nil
}

type StaffArgs struct {
	Barcode   string
	FirstName string
	LastName  string
	PassHash  []byte
}

func (r *Registration) CompleteStaffRegistration(args StaffArgs) error {
	if r.status != StatusPending {
		return errors.New("registration is not pending")
	}

	r.status = StatusCompleted
	r.AddEvent(&StaffRegistrationCompleted{
		Header:         event.NewEventHeader(),
		RegistrationID: r.id,
		Barcode:        args.Barcode,
		Email:          r.email,
		FirstName:      args.FirstName,
		LastName:       args.LastName,
		PassHash:       args.PassHash,
	})

	return nil
}

func (r *Registration) ResendCode() error {
	if r.status != StatusPending {
		return errors.New("can only resend for pending registrations")
	}
	if !r.resendTimeout.IsZero() && !time.Now().After(r.resendTimeout) {
		return fmt.Errorf("cannot resend code yet, please wait until %s", r.resendTimeout)
	}

	code, err := generateCode()
	if err != nil {
		return fmt.Errorf("failed to generate new verification code: %w", err)
	}

	r.verificationCode = code
	r.codeExpiresAt = time.Now().UTC().Add(10 * time.Minute)
	r.codeAttempts = 0
	r.updatedAt = time.Now().UTC()

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

func generateCode() (string, error) {
	code, err := randcode.GenerateAlphaNumericCode(6)
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
