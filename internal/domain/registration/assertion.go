package registration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type RegistrationAssertion struct {
	registration *Registration
}

func NewRegistrationAssertion(reg *Registration) *RegistrationAssertion {
	return &RegistrationAssertion{registration: reg}
}

func (ra *RegistrationAssertion) AssertStatus(t *testing.T, expected Status) *RegistrationAssertion {
	t.Helper()
	assert.Equal(t, expected, ra.registration.status, "Expected registration status to be %s, got %s", expected, ra.registration.status)
	return ra
}

func (ra *RegistrationAssertion) AssertEmail(t *testing.T, expected string) *RegistrationAssertion {
	t.Helper()
	assert.Equal(t, expected, ra.registration.email, "Expected registration email to be %s, got %s", expected, ra.registration.email)
	return ra
}

func (ra *RegistrationAssertion) AssertVerificationCode(t *testing.T, expected string) *RegistrationAssertion {
	t.Helper()
	assert.Equal(
		t,
		expected,
		ra.registration.verificationCode,
		"Expected registration verification code to be %s, got %s",
		expected,
		ra.registration.verificationCode,
	)
	return ra
}

func (ra *RegistrationAssertion) AssertVerificationCodeNotEmpty(t *testing.T) *RegistrationAssertion {
	t.Helper()
	assert.NotEmpty(t, ra.registration.verificationCode, "Expected registration verification code to not be empty")
	return ra
}

func (ra *RegistrationAssertion) AssertVerificationCodeIsNot(t *testing.T, expected string) *RegistrationAssertion {
	t.Helper()
	assert.NotEqual(
		t,
		expected,
		ra.registration.verificationCode,
		"Expected registration verification code to not be %s, got %s",
		expected,
		ra.registration.verificationCode,
	)
	return ra
}

func (ra *RegistrationAssertion) AssertCodeAttempts(t *testing.T, expected int8) *RegistrationAssertion {
	t.Helper()
	assert.Equal(
		t,
		expected,
		ra.registration.codeAttempts,
		"Expected registration code attempts to be %d, got %d",
		expected,
		ra.registration.codeAttempts,
	)
	return ra
}

func (ra *RegistrationAssertion) AssertCodeExpiresAt(t *testing.T, expected time.Time) *RegistrationAssertion {
	t.Helper()
	assert.WithinDuration(
		t,
		expected,
		ra.registration.codeExpiresAt,
		1*time.Second,
		"Expected registration code expires at to be within 1 second of %s, got %s",
		expected,
		ra.registration.codeExpiresAt,
	)
	return ra
}

func (ra *RegistrationAssertion) AssertResendTimeout(t *testing.T, expected time.Time) *RegistrationAssertion {
	t.Helper()
	assert.WithinDuration(
		t,
		expected,
		ra.registration.resendTimeout,
		1*time.Second,
		"Expected registration resend timeout to be within 1 second of %s, got %s",
		expected,
		ra.registration.resendTimeout,
	)
	return ra
}

func (ra *RegistrationAssertion) AssertEventsCount(t *testing.T, expected int) *RegistrationAssertion {
	t.Helper()
	events := ra.registration.GetUncommittedEvents()
	assert.Len(t, events, expected, "Expected %d uncommitted events, got %d", expected, len(events))
	return ra
}

func (ra *RegistrationAssertion) AssertNoEvents(t *testing.T) *RegistrationAssertion {
	t.Helper()
	events := ra.registration.GetUncommittedEvents()
	assert.Empty(t, events, "Expected no uncommitted events, got %d", len(events))
	return ra
}

func (ra *RegistrationAssertion) AssertEventExists(t *testing.T, eventType string) *RegistrationAssertion {
	t.Helper()
	events := ra.registration.GetUncommittedEvents()
	for _, ev := range events {
		if fmt.Sprintf("%T", ev) == eventType {
			return ra
		}
	}
	t.Errorf("Expected event of type %s to exist, but it does not", eventType)
	return ra
}
