package email

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"

	"github.com/ARUMANDESU/ucms/pkg/env"
	"github.com/ARUMANDESU/ucms/pkg/randcode"
)

var emailRx = regexp.MustCompile(
	`^[a-zA-Z0-9._%+\-]+@` + // local part
		`(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.)+` + // labels
		`[A-Za-z]{2,63}$`) // TLD

const (
	ResendTimeout = 1 * time.Minute
	ExpiresAt     = 10 * time.Minute
)

type EmailVerificationCode struct {
	email         string
	code          string
	isUsed        bool
	resendTimeout time.Time
	expiresAt     time.Time
	createdAt     time.Time
	updatedAt     time.Time
}

func NewEmailVerificationCode(email string, mode env.Mode) (*EmailVerificationCode, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}
	if len(email) > 254 {
		return nil, errors.New("email exceeds maximum length of 254 characters")
	}
	if !emailRx.MatchString(email) {
		return nil, fmt.Errorf("invalid email format: %s", email)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, fmt.Errorf("invalid email format: %w", err)
	}
	if (mode == env.Dev || mode == env.Prod) && !hasRealTLD(email) {
		return nil, fmt.Errorf("email must have a real top-level domain (TLD) in %s mode", mode)
	}

	code, err := randcode.GenerateAlphaNumericCode(6)
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification code: %w", err)
	}

	now := time.Now().UTC()

	return &EmailVerificationCode{
		email:         email,
		code:          code,
		isUsed:        false,
		resendTimeout: now.Add(ResendTimeout),
		expiresAt:     now.Add(ExpiresAt),
		createdAt:     now,
		updatedAt:     now,
	}, nil
}

type RehydrateEmailVerificationCodeArgs struct {
	Email         string
	Code          string
	IsUsed        bool
	ResendTimeout time.Time
	ExpiresAt     time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func RehydrateEmailVerificationCode(args RehydrateEmailVerificationCodeArgs) *EmailVerificationCode {
	return &EmailVerificationCode{
		email:         args.Email,
		code:          args.Code,
		isUsed:        args.IsUsed,
		resendTimeout: args.ResendTimeout,
		expiresAt:     args.ExpiresAt,
		createdAt:     args.CreatedAt,
		updatedAt:     args.UpdatedAt,
	}
}

func (ev *EmailVerificationCode) Email() string {
	if ev == nil {
		return ""
	}
	return ev.email
}

func (ev *EmailVerificationCode) Code() string {
	if ev == nil {
		return ""
	}
	return ev.code
}

func (ev *EmailVerificationCode) IsUsed() bool {
	if ev == nil {
		return false
	}
	return ev.isUsed
}

func (ev *EmailVerificationCode) ResendTimeout() time.Time {
	if ev == nil {
		return time.Time{}
	}
	return ev.resendTimeout
}

func (ev *EmailVerificationCode) ExpiresAt() time.Time {
	if ev == nil {
		return time.Time{}
	}
	return ev.expiresAt
}

func (ev *EmailVerificationCode) CreatedAt() time.Time {
	if ev == nil {
		return time.Time{}
	}
	return ev.createdAt
}

func (ev *EmailVerificationCode) UpdatedAt() time.Time {
	if ev == nil {
		return time.Time{}
	}
	return ev.updatedAt
}

func (ev *EmailVerificationCode) IsExpired() bool {
	if ev == nil || ev.expiresAt.IsZero() {
		return true
	}
	return time.Now().After(ev.expiresAt)
}

func (ev *EmailVerificationCode) CanResend() bool {
	if ev == nil || ev.resendTimeout.IsZero() {
		return false
	}
	return time.Now().After(ev.resendTimeout)
}

func (ev *EmailVerificationCode) MarkAsUsed() error {
	if ev == nil {
		return errors.New("email verification code is nil")
	}
	if ev.isUsed {
		return nil
	}
	if ev.IsExpired() {
		return errors.New("email verification code is expired")
	}

	ev.isUsed = true
	ev.updatedAt = time.Now().UTC()
	return nil
}

func (ev *EmailVerificationCode) ReSend() error {
	if ev == nil {
		return errors.New("email verification code is nil")
	}
	if !ev.CanResend() {
		return fmt.Errorf("cannot resend email verification code yet, please wait until %s", ev.resendTimeout)
	}

	code, err := randcode.GenerateAlphaNumericCode(6)
	if err != nil {
		return fmt.Errorf("failed to generate new verification code: %w", err)
	}

	ev.code = code
	ev.isUsed = false
	ev.resendTimeout = time.Now().UTC().Add(ResendTimeout)
	ev.expiresAt = time.Now().UTC().Add(ExpiresAt)
	ev.updatedAt = time.Now().UTC()
	return nil
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
