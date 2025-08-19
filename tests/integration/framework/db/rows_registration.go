package db

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
)

type RegistrationRow struct {
	ID               uuid.UUID
	Email            string
	Status           string
	VerificationCode string
	CodeAttempts     int16
	CodeExpiresAt    time.Time
	ResendTimeout    time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type RegistrationAssertion struct {
	row RegistrationRow
	t   *testing.T
	db  *Helper
}

func (a *RegistrationAssertion) AssertStatus(expected registration.Status) *RegistrationAssertion {
	a.t.Helper()
	assert.Equal(a.t, string(expected), a.row.Status, "unexpected registration status")
	return a
}

func (a *RegistrationAssertion) EventuallyHasStatus(expected registration.Status) *RegistrationAssertion {
	a.t.Helper()
	// time.Sleep(time.Hour)
	assert.Eventually(a.t, func() bool {
		return a.row.Status == string(expected)
	}, 5*time.Second, 100*time.Millisecond, "expected registration status to eventually be %s", expected)
	return a
}

func (a *RegistrationAssertion) AssertVerificationCode() *RegistrationAssertion {
	a.t.Helper()
	assert.NotEmpty(a.t, a.row.VerificationCode, "expected verification code to be set")
	return a
}

func (a *RegistrationAssertion) AssertCodeAttempts(expected int) *RegistrationAssertion {
	a.t.Helper()
	assert.Equal(a.t, expected, int(a.row.CodeAttempts), "unexpected code attempts")
	return a
}

func (a *RegistrationAssertion) AssertIsNotExpired() *RegistrationAssertion {
	a.t.Helper()
	assert.True(a.t, a.row.CodeExpiresAt.After(time.Now()), "registration code is expired")
	return a
}

func (a *RegistrationAssertion) AssertIsExpired() *RegistrationAssertion {
	a.t.Helper()
	assert.True(a.t, a.row.CodeExpiresAt.Before(time.Now()), "registration code is not expired")
	return a
}

func (a *RegistrationAssertion) GetVerificationCode() string {
	return a.row.VerificationCode
}

func (a *RegistrationAssertion) GetID() registration.ID {
	return registration.ID(a.row.ID)
}
