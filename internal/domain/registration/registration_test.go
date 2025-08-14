package registration

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

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
			errorType:   ErrEmptyEmail,
		},
		{
			name:        "email too long",
			email:       "a" + strings.Repeat("b", MaxEmailLength-2) + "@example.com",
			mode:        env.Test,
			expectError: true,
			errorType:   ErrEmailExceedsMaxLength,
		},
		{
			name:        "invalid email format - no @",
			email:       "notanemail",
			mode:        env.Test,
			expectError: true,
			errorType:   ErrInvalidEmailFormat,
		},
		{
			name:        "invalid email format - no domain",
			email:       "user@",
			mode:        env.Test,
			expectError: true,
			errorType:   ErrInvalidEmailFormat,
		},
		{
			name:        "invalid email format - no TLD",
			email:       "user@domain",
			mode:        env.Test,
			expectError: true,
			errorType:   ErrInvalidEmailFormat,
		},
		{
			name:        "localhost email in dev mode",
			email:       "test@localhost",
			mode:        env.Dev,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg, err := NewRegistration(tt.email, tt.mode)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, reg)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
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

func TestRegistration_CompleteStudentRegistration(t *testing.T) {
	t.Run("successful completion", func(t *testing.T) {
		reg := validRegistration(t)

		args := StudentArgs{
			Barcode:          "STU123456",
			FirstName:        "John",
			LastName:         "Doe",
			Password:         "H4rdP@ssw0rd",
			GroupID:          uuid.New(),
			VerificationCode: reg.verificationCode,
		}

		err := reg.CompleteStudentRegistration(args)
		require.NoError(t, err)
		assert.Equal(t, StatusCompleted, reg.status)

		events := reg.GetUncommittedEvents()
		assert.Len(t, events, 2)
		completedEvent, ok := events[1].(*StudentRegistrationCompleted)
		require.True(t, ok)
		assert.Equal(t, reg.id, completedEvent.RegistrationID)
		assert.Equal(t, args.Barcode, completedEvent.Barcode)
		assert.Equal(t, reg.email, completedEvent.Email)
		assert.Equal(t, args.FirstName, completedEvent.FirstName)
		assert.Equal(t, args.LastName, completedEvent.LastName)
		err = bcrypt.CompareHashAndPassword(completedEvent.PassHash, []byte(args.Password))
		assert.NoError(t, err)
		assert.Equal(t, args.GroupID, completedEvent.GroupID)
	})

	t.Run("already verified", func(t *testing.T) {
		reg := validRegistration(t)

		err := reg.VerifyCode(reg.verificationCode)
		require.NoError(t, err)
		reg.MarkEventsAsCommitted()

		args := StudentArgs{
			Barcode:          "STU123456",
			FirstName:        "John",
			LastName:         "Doe",
			Password:         "H4rdP@ssw0rd",
			GroupID:          uuid.New(),
			VerificationCode: reg.verificationCode,
		}

		err = reg.CompleteStudentRegistration(args)
		require.NoError(t, err)
		assert.Equal(t, StatusCompleted, reg.status)

		events := reg.GetUncommittedEvents()
		assert.Len(t, events, 1)
		completedEvent, ok := events[0].(*StudentRegistrationCompleted)
		require.True(t, ok)
		assert.Equal(t, reg.id, completedEvent.RegistrationID)
		assert.Equal(t, args.Barcode, completedEvent.Barcode)
		assert.Equal(t, reg.email, completedEvent.Email)
		assert.Equal(t, args.FirstName, completedEvent.FirstName)
		assert.Equal(t, args.LastName, completedEvent.LastName)
		err = bcrypt.CompareHashAndPassword(completedEvent.PassHash, []byte(args.Password))
		assert.NoError(t, err)
		assert.Equal(t, args.GroupID, completedEvent.GroupID)
	})

	t.Run("not pending status", func(t *testing.T) {
		reg := validRegistration(t)

		reg.status = StatusExpired

		args := StudentArgs{
			VerificationCode: reg.verificationCode,
			Barcode:          "STU123456",
			FirstName:        "John",
			LastName:         "Doe",
			Password:         "H4rdP@ssw0rd",
			GroupID:          uuid.New(),
		}

		err := reg.CompleteStudentRegistration(args)
		assert.ErrorIs(t, err, ErrInvalidStatus)
	})
}

func TestRegistration_CompleteStaffRegistration(t *testing.T) {
	t.Run("successful completion", func(t *testing.T) {
		reg := validRegistration(t)

		args := StaffArgs{
			VerificationCode: reg.verificationCode,
			Barcode:          "STAFF123",
			FirstName:        "Jane",
			LastName:         "Smith",
			Password:         "H4rdP@ssw0rd",
		}

		err := reg.CompleteStaffRegistration(args)
		require.NoError(t, err)
		assert.Equal(t, StatusCompleted, reg.status)

		events := reg.GetUncommittedEvents()
		assert.Len(t, events, 2)
		completedEvent, ok := events[1].(*StaffRegistrationCompleted)
		require.True(t, ok)
		assert.Equal(t, reg.id, completedEvent.RegistrationID)
		assert.Equal(t, args.Barcode, completedEvent.Barcode)
		assert.Equal(t, reg.email, completedEvent.Email)
		assert.Equal(t, args.FirstName, completedEvent.FirstName)
		assert.Equal(t, args.LastName, completedEvent.LastName)
		err = bcrypt.CompareHashAndPassword(completedEvent.PassHash, []byte(args.Password))
		assert.NoError(t, err)
	})

	t.Run("already verified", func(t *testing.T) {
		reg := validRegistration(t)

		err := reg.VerifyCode(reg.verificationCode)
		require.NoError(t, err)
		reg.MarkEventsAsCommitted()

		args := StaffArgs{
			VerificationCode: reg.verificationCode,
			Barcode:          "STAFF123",
			FirstName:        "Jane",
			LastName:         "Smith",
			Password:         "H4rdP@ssw0rd",
		}
		require.NoError(t, err)

		err = reg.CompleteStaffRegistration(args)
		require.NoError(t, err)
		assert.Equal(t, StatusCompleted, reg.status)

		events := reg.GetUncommittedEvents()
		assert.Len(t, events, 1)
		completedEvent, ok := events[0].(*StaffRegistrationCompleted)
		require.True(t, ok)
		assert.Equal(t, reg.id, completedEvent.RegistrationID)
		assert.Equal(t, args.Barcode, completedEvent.Barcode)
		assert.Equal(t, reg.email, completedEvent.Email)
		assert.Equal(t, args.FirstName, completedEvent.FirstName)
		assert.Equal(t, args.LastName, completedEvent.LastName)
		err = bcrypt.CompareHashAndPassword(completedEvent.PassHash, []byte(args.Password))
		assert.NoError(t, err)
	})

	t.Run("not pending status", func(t *testing.T) {
		reg := validRegistration(t)

		reg.status = StatusExpired

		args := StaffArgs{
			VerificationCode: reg.verificationCode,
			Barcode:          "STAFF123",
			FirstName:        "Jane",
			LastName:         "Smith",
			Password:         "H4rdP@ssw0rd",
		}

		err := reg.CompleteStaffRegistration(args)
		assert.ErrorIs(t, err, ErrInvalidStatus)
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

func TestHasRealTLD(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		{
			name:     "gmail.com",
			email:    "user@gmail.com",
			expected: true,
		},
		{
			name:     "yahoo.com",
			email:    "user@yahoo.com",
			expected: true,
		},
		{
			name:     "localhost",
			email:    "user@localhost",
			expected: false,
		},
		{
			name:     "internal",
			email:    "user@internal",
			expected: false,
		},
		{
			name:     "example.local",
			email:    "user@example.local",
			expected: false,
		},
		{
			name:     "invalid email",
			email:    "notanemail",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasRealTLD(tt.email)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEmailValidationConstants(t *testing.T) {
	assert.Equal(t, 1*time.Minute, ResendTimeout)
	assert.Equal(t, 10*time.Minute, ExpiresAt)
	assert.Equal(t, 254, MaxEmailLength)
}

func TestEmailRegex(t *testing.T) {
	validEmails := []string{
		"test@example.com",
		"user.name@domain.com",
		"user+tag@example.org",
		"user-name@example-domain.com",
	}

	invalidEmails := []string{
		"test",
		"test@",
		"@example.com",
		"test@example",
		"test@example..com",
	}

	for _, email := range validEmails {
		t.Run("valid_"+email, func(t *testing.T) {
			assert.True(t, emailRx.MatchString(email), "Expected %s to be valid", email)
		})
	}

	for _, email := range invalidEmails {
		t.Run("invalid_"+email, func(t *testing.T) {
			assert.False(t, emailRx.MatchString(email), "Expected %s to be invalid", email)
		})
	}
}

func TestRegistrationWorkflow(t *testing.T) {
	t.Run("complete student registration workflow", func(t *testing.T) {
		// 1. Create registration
		reg, err := NewRegistration("student@example.com", env.Test)
		require.NoError(t, err)
		reg.MarkEventsAsCommitted()

		regAss := NewRegistrationAssertion(reg).
			AssertStatus(t, StatusPending)

		// 2. Verify email
		err = reg.VerifyCode(reg.verificationCode)
		require.NoError(t, err)
		regAss.
			AssertStatus(t, StatusVerified).
			AssertEmail(t, reg.email).
			AssertVerificationCode(t, reg.verificationCode).
			AssertCodeAttempts(t, 0)

		// 3. Complete student registration
		args := StudentArgs{
            VerificationCode: reg.verificationCode,
			Barcode:   "ST123456",
			FirstName: "John",
			LastName:  "Doe",
			Password:  "H4rdP@ssw0rd",
			GroupID:   uuid.New(),
		}

		err = reg.CompleteStudentRegistration(args)
		require.NoError(t, err)
		regAss.
			AssertStatus(t, StatusCompleted).
			AssertEmail(t, reg.email).
			AssertVerificationCodeNotEmpty(t).
			AssertCodeAttempts(t, 0).
			AssertCodeExpiresAt(t, time.Now().Add(ExpiresAt)).
			AssertResendTimeout(t, time.Now().Add(ResendTimeout)).
			AssertEventsCount(t, 2)

		events := reg.GetUncommittedEvents()
		assert.Len(t, events, 2)
		completedEvent, ok := events[1].(*StudentRegistrationCompleted)
		require.True(t, ok)
		assert.Equal(t, reg.id, completedEvent.RegistrationID)
		assert.Equal(t, args.Barcode, completedEvent.Barcode)
		assert.Equal(t, reg.email, completedEvent.Email)
		assert.Equal(t, args.FirstName, completedEvent.FirstName)
		assert.Equal(t, args.LastName, completedEvent.LastName)
		err = bcrypt.CompareHashAndPassword(completedEvent.PassHash, []byte(args.Password))
		assert.NoError(t, err)
		assert.Equal(t, args.GroupID, completedEvent.GroupID)
	})

	t.Run("complete staff registration workflow", func(t *testing.T) {
		// 1. Create registration
		reg, err := NewRegistration("staff@example.com", env.Test)
		require.NoError(t, err)
		assert.Equal(t, StatusPending, reg.status)

		// 2. Verify email
		err = reg.VerifyCode(reg.verificationCode)
		require.NoError(t, err)

		// 3. Complete staff registration
		args := StaffArgs{
            VerificationCode: reg.verificationCode,
			Barcode:   "STAFF123",
			FirstName: "Jane",
			LastName:  "Smith",
			Password:  "H4rdP@ssw0rd",
		}

		err = reg.CompleteStaffRegistration(args)
		require.NoError(t, err)
		assert.Equal(t, StatusCompleted, reg.status)
		assert.True(t, reg.IsCompleted())
	})
}

func validRegistration(t *testing.T) *Registration {
	reg, err := NewRegistration("test@example.com", env.Test)
	require.NoError(t, err, "Failed to create valid registration")
	reg.MarkEventsAsCommitted()
	return reg
}
