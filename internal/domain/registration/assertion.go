package registration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type RegistrationAssertion struct {
	Registration *Registration
}

func NewRegistrationAssertion(reg *Registration) *RegistrationAssertion {
	return &RegistrationAssertion{Registration: reg}
}

func (ra *RegistrationAssertion) AssertStatus(t *testing.T, expected Status) *RegistrationAssertion {
	t.Helper()
	assert.Equal(t, expected, ra.Registration.status, "Expected registration status to be %s, got %s", expected, ra.Registration.status)
	return ra
}

func (ra *RegistrationAssertion) AssertEmail(t *testing.T, expected string) *RegistrationAssertion {
	t.Helper()
	assert.Equal(t, expected, ra.Registration.email, "Expected registration email to be %s, got %s", expected, ra.Registration.email)
	return ra
}

func (ra *RegistrationAssertion) AssertVerificationCode(t *testing.T, expected string) *RegistrationAssertion {
	t.Helper()
	assert.Equal(
		t,
		expected,
		ra.Registration.verificationCode,
		"Expected registration verification code to be %s, got %s",
		expected,
		ra.Registration.verificationCode,
	)
	return ra
}

func (ra *RegistrationAssertion) AssertVerificationCodeNotEmpty(t *testing.T) *RegistrationAssertion {
	t.Helper()
	assert.NotEmpty(t, ra.Registration.verificationCode, "Expected registration verification code to not be empty")
	return ra
}

func (ra *RegistrationAssertion) AssertVerificationCodeIsNot(t *testing.T, expected string) *RegistrationAssertion {
	t.Helper()
	assert.NotEqual(
		t,
		expected,
		ra.Registration.verificationCode,
		"Expected registration verification code to not be %s, got %s",
		expected,
		ra.Registration.verificationCode,
	)
	return ra
}

func (ra *RegistrationAssertion) AssertCodeAttempts(t *testing.T, expected int8) *RegistrationAssertion {
	t.Helper()
	assert.Equal(
		t,
		expected,
		ra.Registration.codeAttempts,
		"Expected registration code attempts to be %d, got %d",
		expected,
		ra.Registration.codeAttempts,
	)
	return ra
}

func (ra *RegistrationAssertion) AssertCodeExpiresAt(t *testing.T, expected time.Time) *RegistrationAssertion {
	t.Helper()
	assert.WithinDuration(
		t,
		expected,
		ra.Registration.codeExpiresAt,
		1*time.Second,
		"Expected registration code expires at to be within 1 second of %s, got %s",
		expected,
		ra.Registration.codeExpiresAt,
	)
	return ra
}

func (ra *RegistrationAssertion) AssertResendTimeout(t *testing.T, expected time.Time) *RegistrationAssertion {
	t.Helper()
	assert.WithinDuration(
		t,
		expected,
		ra.Registration.resendTimeout,
		1*time.Second,
		"Expected registration resend timeout to be within 1 second of %s, got %s",
		expected,
		ra.Registration.resendTimeout,
	)
	return ra
}

func (ra *RegistrationAssertion) AssertResendNotAvailable(t *testing.T) *RegistrationAssertion {
	t.Helper()
	assert.True(
		t,
		// check if resend timeout is in the future, because if it is, then resend is not available
		ra.Registration.resendTimeout.After(time.Now()),
		"Expected registration resend timeout to be in the future, got %s; current time is %s",
		ra.Registration.resendTimeout,
		time.Now(),
	)
	return ra
}

func (ra *RegistrationAssertion) AssertIsNotExpired(t *testing.T) *RegistrationAssertion {
	t.Helper()
	assert.True(t, ra.Registration.codeExpiresAt.After(time.Now()),
		"Expected registration code to not be expired, but it is; code expires at %s, current time is %s",
		ra.Registration.codeExpiresAt,
		time.Now(),
	)
	return ra
}

func (ra *RegistrationAssertion) AssertEventsCount(t *testing.T, expected int) *RegistrationAssertion {
	t.Helper()
	events := ra.Registration.GetUncommittedEvents()
	assert.Len(t, events, expected, "Expected %d uncommitted events, got %d", expected, len(events))
	return ra
}

func (ra *RegistrationAssertion) AssertNoEvents(t *testing.T) *RegistrationAssertion {
	t.Helper()
	events := ra.Registration.GetUncommittedEvents()
	assert.Empty(t, events, "Expected no uncommitted events, got %d", len(events))
	return ra
}

func (ra *RegistrationAssertion) AssertEventExists(t *testing.T, eventType string) *RegistrationAssertion {
	t.Helper()
	events := ra.Registration.GetUncommittedEvents()
	for _, ev := range events {
		if fmt.Sprintf("%T", ev) == eventType {
			return ra
		}
	}
	t.Errorf("Expected event of type %s to exist, but it does not", eventType)
	return ra
}

type RegistrationStartedAssertion struct {
	event *RegistrationStarted
}

func NewRegistrationStartedAssertion(event *RegistrationStarted) *RegistrationStartedAssertion {
	return &RegistrationStartedAssertion{event: event}
}

func (rsa *RegistrationStartedAssertion) AssertRegistrationID(t *testing.T, expected ID) *RegistrationStartedAssertion {
	t.Helper()
	assert.Equal(t, expected, rsa.event.RegistrationID, "Expected registration ID to be %s, got %s", expected, rsa.event.RegistrationID)
	return rsa
}

func (rsa *RegistrationStartedAssertion) AssertEmail(t *testing.T, expected string) *RegistrationStartedAssertion {
	t.Helper()
	assert.Equal(t, expected, rsa.event.Email, "Expected registration email to be %s, got %s", expected, rsa.event.Email)
	return rsa
}

func (rsa *RegistrationStartedAssertion) AssertVerificationCode(t *testing.T, expected string) *RegistrationStartedAssertion {
	t.Helper()
	assert.Equal(
		t,
		expected,
		rsa.event.VerificationCode,
		"Expected registration verification code to be %s, got %s",
		expected,
		rsa.event.VerificationCode,
	)
	return rsa
}

func (rsa *RegistrationStartedAssertion) AssertRegistrationIDNotEmpty(t *testing.T) *RegistrationStartedAssertion {
	t.Helper()
	assert.NotEmpty(t, rsa.event.RegistrationID, "Expected registration ID to not be empty")
	return rsa
}

func (rsa *RegistrationStartedAssertion) AssertVerificationCodeNotEmpty(t *testing.T) *RegistrationStartedAssertion {
	t.Helper()
	assert.NotEmpty(t, rsa.event.VerificationCode, "Expected registration verification code to not be empty")
	return rsa
}

type VerificationCodeResentAssertion struct {
	event *VerificationCodeResent
}

func NewVerificationCodeSentAssertion(event *VerificationCodeResent) *VerificationCodeResentAssertion {
	return &VerificationCodeResentAssertion{event: event}
}

func (vsa *VerificationCodeResentAssertion) AssertRegistrationID(t *testing.T, expected ID) *VerificationCodeResentAssertion {
	t.Helper()
	assert.Equal(t, expected, vsa.event.RegistrationID, "Expected registration ID to be %s, got %s", expected, vsa.event.RegistrationID)
	return vsa
}

func (vsa *VerificationCodeResentAssertion) AssertEmail(t *testing.T, expected string) *VerificationCodeResentAssertion {
	t.Helper()
	assert.Equal(t, expected, vsa.event.Email, "Expected registration email to be %s, got %s", expected, vsa.event.Email)
	return vsa
}

func (vsa *VerificationCodeResentAssertion) AssertVerificationCode(t *testing.T, expected string) *VerificationCodeResentAssertion {
	t.Helper()
	assert.Equal(
		t,
		expected,
		vsa.event.VerificationCode,
		"Expected registration verification code to be %s, got %s",
		expected,
		vsa.event.VerificationCode,
	)
	return vsa
}

func (vsa *VerificationCodeResentAssertion) AssertVerificationCodeNotEqual(t *testing.T, expected string) *VerificationCodeResentAssertion {
	t.Helper()
	assert.NotEqual(
		t,
		expected,
		vsa.event.VerificationCode,
		"Expected registration verification code not to be %s, got %s",
		expected,
		vsa.event.VerificationCode,
	)
	return vsa
}

func (vsa *VerificationCodeResentAssertion) AssertVerificationCodeNotEmpty(t *testing.T) *VerificationCodeResentAssertion {
	t.Helper()
	assert.NotEmpty(t, vsa.event.VerificationCode, "Expected registration verification code to not be empty")
	return vsa
}
