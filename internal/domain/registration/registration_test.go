package registration

import (
	"strings"
	"testing"
	"time"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/pkg/env"
)

func TestNewRegistration(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		mode        env.Mode
		expectError bool
		errorType   error
	}{
		{
			name:        "valid email in test mode",
			email:       "test@example.com",
			mode:        env.Test,
			expectError: false,
		},
		{
			name:        "valid email in dev mode",
			email:       "user@gmail.com",
			mode:        env.Dev,
			expectError: false,
		},
		{
			name:        "valid email in prod mode",
			email:       "user@gmail.com",
			mode:        env.Prod,
			expectError: false,
		},
		{
			name:        "empty email",
			email:       "",
			mode:        env.Test,
			expectError: true,
			errorType:   validation.ErrEmpty,
		},
		{
			name:        "email too long",
			email:       "a" + strings.Repeat("b", 255) + "@example.com", // 256 characters
			mode:        env.Test,
			expectError: true,
			errorType:   is.ErrEmail,
		},
		{
			name:        "invalid email format - no @",
			email:       "notanemail",
			mode:        env.Test,
			expectError: true,
			errorType:   is.ErrEmail,
		},
		{
			name:        "invalid email format - no domain",
			email:       "user@",
			mode:        env.Test,
			expectError: true,
			errorType:   is.ErrEmail,
		},
		{
			name:        "invalid email format - no TLD",
			email:       "user@domain",
			mode:        env.Test,
			expectError: true,
			errorType:   is.ErrEmail,
		},
		{
			name:        "localhost email in dev mode",
			email:       "test@localhost",
			mode:        env.Dev,
			expectError: false,
			errorType:   is.ErrEmail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg, err := NewRegistration(tt.email, tt.mode)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, reg)
				if tt.errorType != nil {
					assert.ErrorAs(t, err, &tt.errorType)
				} else {
					assert.ErrorIs(t, err, validation.ErrEmpty)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, reg)

				NewRegistrationAssertion(reg).
					AssertStatus(t, StatusPending).
					AssertEmail(t, tt.email).
					AssertVerificationCodeNotEmpty(t).
					AssertCodeAttempts(t, 0).
					AssertCodeExpiresAt(t, time.Now().Add(ExpiresAt)).
					AssertResendTimeout(t, time.Now().Add(ResendTimeout)).
					AssertEventsCount(t, 1)

				events := reg.GetUncommittedEvents()
				assert.Len(t, events, 1)
				startedEvent, ok := events[0].(*RegistrationStarted)
				assert.True(t, ok)
				assert.Equal(t, reg.id, startedEvent.RegistrationID)
				assert.Equal(t, tt.email, startedEvent.Email)
				assert.Equal(t, reg.verificationCode, startedEvent.VerificationCode)
			}
		})
	}
}

func TestRegistration_VerifyCode(t *testing.T) {
	t.Run("successful verification", func(t *testing.T) {
		reg := validRegistration(t)

		err := reg.VerifyCode(reg.verificationCode)
		assert.NoError(t, err)

		NewRegistrationAssertion(reg).
			AssertStatus(t, StatusVerified).
			AssertCodeAttempts(t, 0).
			AssertEventsCount(t, 1)

		events := reg.GetUncommittedEvents()
		assert.Len(t, events, 1)
		verifiedEvent, ok := events[0].(*EmailVerified)
		assert.True(t, ok)
		assert.Equal(t, reg.id, verifiedEvent.RegistrationID)
		assert.Equal(t, reg.email, verifiedEvent.Email)
	})

	t.Run("invalid code", func(t *testing.T) {
		reg := validRegistration(t)

		err := reg.VerifyCode("wrongcode")
		require.ErrorIs(t, err, ErrPersistentVerificationCodeMismatch)

		NewRegistrationAssertion(reg).
			AssertStatus(t, StatusPending).
			AssertCodeAttempts(t, 1).
			AssertEventsCount(t, 0)
	})

	t.Run("too many failed attempts", func(t *testing.T) {
		reg := validRegistration(t)

		for range 3 {
			err := reg.VerifyCode("wrongcode")
			assert.Error(t, err)
		}

		NewRegistrationAssertion(reg).
			AssertStatus(t, StatusExpired).
			AssertCodeAttempts(t, MaxVerificationCodeAttempts).
			AssertEventsCount(t, 1)

		events := reg.GetUncommittedEvents()
		assert.Len(t, events, 1)
		failedEvent, ok := events[0].(*RegistrationFailed)
		assert.True(t, ok)
		assert.Equal(t, reg.id, failedEvent.RegistrationID)
		assert.Equal(t, "too many failed attempts", failedEvent.Reason)
	})

	t.Run("expired code", func(t *testing.T) {
		reg := validRegistration(t)

		reg.codeExpiresAt = time.Now().Add(-1 * time.Minute)

		err := reg.VerifyCode(reg.verificationCode)
		assert.ErrorIs(t, err, ErrCodeExpired)
		assert.Equal(t, StatusExpired, reg.status)
	})

	t.Run("not pending status", func(t *testing.T) {
		reg := validRegistration(t)

		reg.status = StatusCompleted

		err := reg.VerifyCode(reg.verificationCode)
		assert.ErrorIs(t, err, ErrInvalidStatus)
	})
}

func TestRegistration_CheckCode(t *testing.T) {
	t.Run("successful check", func(t *testing.T) {
		reg := validRegistration(t)
		reg.status = StatusVerified

		err := reg.CheckCode(reg.verificationCode)
		assert.NoError(t, err)

		NewRegistrationAssertion(reg).
			AssertStatus(t, reg.status).
			AssertCodeAttempts(t, 0).
			AssertEventsCount(t, 0)
	})

	t.Run("invalid code", func(t *testing.T) {
		reg := validRegistration(t)
		reg.status = StatusVerified

		err := reg.CheckCode("wrongcode")
		assert.ErrorIs(t, err, ErrInvalidVerificationCode)

		NewRegistrationAssertion(reg).
			AssertStatus(t, reg.status).
			AssertCodeAttempts(t, 0).
			AssertEventsCount(t, 0)
	})

	t.Run("expired code", func(t *testing.T) {
		reg := validRegistration(t)
		reg.status = StatusVerified

		reg.codeExpiresAt = time.Now().Add(-1 * time.Minute)

		err := reg.CheckCode(reg.verificationCode)
		assert.ErrorIs(t, err, ErrCodeExpired)
	})

	t.Run("not verified status", func(t *testing.T) {
		reg := validRegistration(t)

		reg.status = StatusPending

		err := reg.CheckCode(reg.verificationCode)
		assert.ErrorIs(t, err, ErrVerifyFirst)
	})
}

func TestRegistration_ResendCode(t *testing.T) {
	t.Run("successful resend after timeout", func(t *testing.T) {
		reg := validRegistration(t)
		reg.resendTimeout = time.Now().Add(-1 * time.Minute)
		originalCode := reg.verificationCode

		err := reg.ResendCode()
		require.NoError(t, err)
		NewRegistrationAssertion(reg).
			AssertStatus(t, StatusPending).
			AssertEmail(t, reg.email).
			AssertVerificationCodeIsNot(t, originalCode).
			AssertCodeAttempts(t, 0).
			AssertResendNotAvailable(t).
			AssertEventsCount(t, 1)

		events := reg.GetUncommittedEvents()
		assert.Len(t, events, 1)
		resentEvent, ok := events[0].(*VerificationCodeResent)
		assert.True(t, ok)
		assert.Equal(t, reg.id, resentEvent.RegistrationID)
		assert.Equal(t, reg.email, resentEvent.Email)
		assert.Equal(t, reg.verificationCode, resentEvent.VerificationCode)
	})

	t.Run("resend too early", func(t *testing.T) {
		reg := validRegistration(t)

		err := reg.ResendCode()
		assert.ErrorIs(t, err, ErrWaitUntilResend)
	})
}

func TestRegistration_Complete(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*Registration)
		expectError bool
		errorType   error
	}{
		{
			name:        "successful completion",
			setup:       func(reg *Registration) { reg.status = StatusVerified },
			expectError: false,
		},
		{
			name:        "not verified status",
			setup:       func(reg *Registration) { reg.status = StatusPending },
			expectError: true,
			errorType:   ErrInvalidStatus,
		},
		{
			name:        "already completed",
			setup:       func(reg *Registration) { reg.status = StatusCompleted },
			expectError: false,
		},
		{
			name:        "expired registration",
			setup:       func(reg *Registration) { reg.status = StatusExpired },
			expectError: true,
			errorType:   ErrInvalidStatus,
		},
		{
			name:        "nil registration",
			setup:       func(reg *Registration) {},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reg *Registration
			if tt.name != "nil registration" {
				reg = validRegistration(t)
				tt.setup(reg)
			}

			err := reg.Complete()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorAs(t, err, &tt.errorType)
				}
			} else {
				require.NoError(t, err)
				NewRegistrationAssertion(reg).
					AssertStatus(t, StatusCompleted).
					AssertEmail(t, reg.email).
					AssertVerificationCodeNotEmpty(t).
					AssertCodeAttempts(t, 0).
					AssertResendNotAvailable(t).
					AssertEventsCount(t, 0)
			}
		})
	}
}

func TestRegistration_IsStatus(t *testing.T) {
	reg := validRegistration(t)

	assert.True(t, reg.IsStatus(StatusPending))
	assert.False(t, reg.IsStatus(StatusCompleted))
	assert.False(t, reg.IsStatus(StatusExpired))

	var nilReg *Registration
	assert.False(t, nilReg.IsStatus(StatusPending))
}

func TestRegistration_IsCompleted(t *testing.T) {
	reg := validRegistration(t)

	assert.False(t, reg.IsCompleted())

	reg.status = StatusCompleted
	assert.True(t, reg.IsCompleted())
}

func validRegistration(t *testing.T) *Registration {
	reg, err := NewRegistration("test@example.com", env.Test)
	require.NoError(t, err, "Failed to create valid registration")
	reg.MarkEventsAsCommitted()
	return reg
}
