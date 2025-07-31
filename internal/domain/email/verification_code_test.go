package email

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/pkg/env"
)

func TestNewEmailVerificationCode(t *testing.T) {
	t.Parallel()
	ok := func(want string) func(*testing.T, *EmailVerificationCode, error) {
		return func(t *testing.T, code *EmailVerificationCode, err error) {
			assert.NoError(t, err)
			assert.NotNil(t, code)
			assert.Equal(t, 6, len(code.code))
			assert.Equal(t, want, code.email)
			assert.False(t, code.isUsed)

			now := time.Now().UTC()
			assert.WithinDuration(t, now, code.createdAt, 1*time.Second)
			assert.WithinDuration(t, now.Add(ExpiresAt), code.expiresAt, 1*time.Second)
			assert.WithinDuration(t, now, code.updatedAt, 1*time.Second)
		}
	}
	bad := func(t *testing.T, code *EmailVerificationCode, err error) {
		assert.Error(t, err)
		assert.Nil(t, code)
	}

	tests := []struct {
		name  string
		email string
		mode  env.Mode
		want  func(t *testing.T, code *EmailVerificationCode, err error)
	}{
		// happy paths ----------------------------------------------------
		{
			name:  "valid simple",
			email: "test@gmail.com",
			want:  ok("test@gmail.com"),
		},
		{
			name:  "plus alias",
			email: "john.doe+newsletter@example.com",
			want:  ok("john.doe+newsletter@example.com"),
		},
		{
			name:  "sub‑domain",
			email: "user@mail.dev.example.co.uk",
			want:  ok("user@mail.dev.example.co.uk"),
		},
		{
			name:  "minimal length",
			email: "a@b.co",
			want:  ok("a@b.co"),
		},
		{
			name:  "mixed case allowed",
			email: "User@Example.COM",
			want:  ok("User@Example.COM"),
		},

		// error cases ----------------------------------------------------
		{
			name:  "empty email",
			email: "",
			want:  bad,
		},
		{
			name:  "invalid format",
			email: "invalid-email",
			want:  bad,
		},
		{
			name:  "spaces inside",
			email: "test @gmail.com",
			want:  bad,
		},
		{
			name:  "double at",
			email: "foo@@bar.com",
			want:  bad,
		},
		{
			name:  "consecutive dots",
			email: "foo..bar@example.com",
			want:  bad,
		},
		{
			name:  "no top‑level domain in prod",
			email: "foo@localhost",
			mode:  env.Prod, // TLD is required in production
			want:  bad,
		},
		{
			name:  "no top‑level domain in dev",
			email: "foo@localhost",
			mode:  env.Dev, // TLD is required in development
			want:  bad,
		},
		{
			name:  "trailing dot in domain",
			email: "foo@bar.com.",
			want:  bad,
		},
		{
			name:  "domain starts with dash",
			email: "foo@-example.com",
			want:  bad,
		},
		{
			name:  "too long (>254 chars)",
			email: strings.Repeat("x", 245) + "@example.com", // 245+1+11 = 257
			want:  bad,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, err := NewEmailVerificationCode(tc.email, tc.mode)
			tc.want(t, code, err)
		})
	}
}

func TestEmailVerificationCode_Getter(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		code, err := NewEmailVerificationCode("valid@gmail.com", env.Prod)
		assert.NoError(t, err)
		assert.NotNil(t, code)
		assert.Equal(t, code.email, code.Email())
		assert.Equal(t, code.code, code.Code())
		assert.Equal(t, code.isUsed, code.IsUsed())
		assert.WithinDuration(t, code.resendTimeout, code.ResendTimeout(), 1*time.Second)
		assert.WithinDuration(t, code.expiresAt, code.ExpiresAt(), 1*time.Second)
		assert.WithinDuration(t, code.createdAt, code.CreatedAt(), 1*time.Second)
		assert.WithinDuration(t, code.updatedAt, code.UpdatedAt(), 1*time.Second)

		assert.False(t, code.IsExpired())
		assert.False(t, code.CanResend())
	})

	t.Run("nil receiver", func(t *testing.T) {
		var code *EmailVerificationCode
		assert.Empty(t, code.Email())
		assert.Empty(t, code.Code())
		assert.False(t, code.IsUsed())
		assert.Equal(t, time.Time{}, code.ResendTimeout())
		assert.Equal(t, time.Time{}, code.ExpiresAt())
		assert.Equal(t, time.Time{}, code.CreatedAt())
		assert.Equal(t, time.Time{}, code.UpdatedAt())

		assert.True(t, code.IsExpired())
		assert.False(t, code.CanResend())
	})

	t.Run("empty receiver", func(t *testing.T) {
		code := &EmailVerificationCode{}
		assert.Empty(t, code.Email())
		assert.Empty(t, code.Code())
		assert.False(t, code.IsUsed())
		assert.Equal(t, time.Time{}, code.ResendTimeout())
		assert.Equal(t, time.Time{}, code.ExpiresAt())
		assert.Equal(t, time.Time{}, code.CreatedAt())
		assert.Equal(t, time.Time{}, code.UpdatedAt())

		assert.True(t, code.IsExpired())
		assert.False(t, code.CanResend())
	})
}

func TestEmailVerificationCode_IsExpired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "not expired",
			expiresAt: time.Now().Add(1 * time.Hour),
			want:      false,
		},
		{
			name:      "expired",
			expiresAt: time.Now().Add(-1 * time.Hour),
			want:      true,
		},
		{
			name:      "exactly now",
			expiresAt: time.Now(),
			want:      true,
		},
		{
			name:      "zero time",
			expiresAt: time.Time{},
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := &EmailVerificationCode{
				expiresAt: tt.expiresAt,
			}
			assert.Equal(t, tt.want, code.IsExpired())
		})
	}
}

func TestEmailVerificationCode_CanResend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		resendTimeout time.Time
		want          bool
	}{
		{
			name:          "can resend",
			resendTimeout: time.Now().Add(-1 * time.Hour),
			want:          true,
		},
		{
			name:          "cannot resend yet",
			resendTimeout: time.Now().Add(1 * time.Hour),
			want:          false,
		},
		{
			name:          "exactly now",
			resendTimeout: time.Now(),
			want:          true,
		},
		{
			name:          "zero time",
			resendTimeout: time.Time{},
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := &EmailVerificationCode{
				resendTimeout: tt.resendTimeout,
			}
			assert.Equal(t, tt.want, code.CanResend())
		})
	}
}

func TestEmailVerificationCode_MarkAsUsed(t *testing.T) {
	t.Parallel()

	t.Run("mark as used", func(t *testing.T) {
		code, err := NewEmailVerificationCode("valid@gmail.com", env.Prod)
		assert.NoError(t, err)
		assert.NotNil(t, code)
		assert.False(t, code.isUsed)
		assert.NoError(t, code.MarkAsUsed())
		assert.True(t, code.isUsed)
		assert.WithinDuration(t, time.Now().UTC(), code.updatedAt, 1*time.Second)
	})

	t.Run("already used", func(t *testing.T) {
		now := time.Now().UTC()
		code := &EmailVerificationCode{
			isUsed:    true,
			updatedAt: now,
		}
		assert.NoError(t, code.MarkAsUsed())
		assert.True(t, code.isUsed)
		assert.WithinDuration(t, now, code.updatedAt, 10*time.Millisecond)
	})

	t.Run("expired code", func(t *testing.T) {
		now := time.Now().UTC()
		code := &EmailVerificationCode{
			isUsed:    false,
			expiresAt: now.Add(-1 * time.Minute),
			updatedAt: now.Add(-10 * time.Minute),
		}
		assert.Error(t, code.MarkAsUsed())
		assert.False(t, code.isUsed)
		assert.WithinDuration(t, now.Add(-10*time.Minute), code.updatedAt, 10*time.Millisecond)
	})

	t.Run("nil receiver", func(t *testing.T) {
		var code *EmailVerificationCode
		err := code.MarkAsUsed()
		assert.Error(t, err)
		assert.EqualError(t, err, "email verification code is nil")
	})

	t.Run("empty receiver", func(t *testing.T) {
		code := &EmailVerificationCode{}
		assert.Error(t, code.MarkAsUsed())
		assert.EqualError(t, code.MarkAsUsed(), "email verification code is expired")
	})
}

func TestEmailVerificationCode_ReSend(t *testing.T) {
	t.Parallel()

	t.Run("resend valid", func(t *testing.T) {
		code, err := NewEmailVerificationCode("valid@gmail.com", env.Prod)
		assert.NoError(t, err)
		assert.NotNil(t, code)
		assert.False(t, code.isUsed)
		code.resendTimeout = time.Now().Add(-1 * time.Second) // Force resend allowed

		prevCode := code.code
		prevUpdatedAt := code.updatedAt
		prevExpiresAt := code.expiresAt
		prevResendTimeout := code.resendTimeout

		require.NoError(t, code.ReSend())
		assert.NotEqual(t, prevCode, code.code, "code should change on resend, previous code: %s, new code: %s", prevCode, code.code)
		assert.False(t, code.isUsed)
		assert.WithinDuration(t, time.Now().UTC(), code.updatedAt, 1*time.Second)
		assert.WithinDuration(t, time.Now().UTC().Add(ExpiresAt), code.expiresAt, 1*time.Second)
		assert.WithinDuration(t, time.Now().UTC().Add(ResendTimeout), code.resendTimeout, 1*time.Second)
		assert.Greater(t, code.updatedAt, prevUpdatedAt, "updatedAt should be in the future")
		assert.Greater(t, code.expiresAt, prevExpiresAt, "expiresAt should be in the future")
		assert.Greater(t, code.resendTimeout, prevResendTimeout, "resendTimeout should be in the future")
	})

	t.Run("resend not allowed", func(t *testing.T) {
		code, err := NewEmailVerificationCode("valid@gmail.com", env.Prod)
		assert.NoError(t, err)
		assert.NotNil(t, code)
		assert.False(t, code.isUsed)

		require.False(t, code.CanResend(), "should not be able to resend yet")
		assert.Error(t, code.ReSend(), "should return error when trying to resend too early")
	})

	t.Run("resend after used", func(t *testing.T) {
		code, err := NewEmailVerificationCode("valid@gmail.com", env.Prod)
		assert.NoError(t, err)
		assert.NotNil(t, code)
		assert.False(t, code.isUsed)

		// Mark as used first
		require.NoError(t, code.MarkAsUsed(), "should mark as used successfully")
		assert.True(t, code.isUsed, "code should be marked as used")
		assert.Error(t, code.ReSend(), "should return error when trying to resend after used")
	})

	t.Run("nil receiver", func(t *testing.T) {
		var code *EmailVerificationCode
		err := code.ReSend()
		assert.Error(t, err)
		assert.EqualError(t, err, "email verification code is nil")
	})

	t.Run("empty receiver", func(t *testing.T) {
		code := &EmailVerificationCode{}
		err := code.ReSend()
		assert.Error(t, err)
		assert.EqualError(t, err, "cannot resend email verification code yet, please wait until "+code.resendTimeout.String())
	})

	t.Run("expired code", func(t *testing.T) {
		now := time.Now().UTC()
		code := &EmailVerificationCode{
			isUsed:        false,
			expiresAt:     now.Add(-1 * time.Minute),
			updatedAt:     now.Add(-10 * time.Minute),
			resendTimeout: now.Add(-1 * time.Minute), // Force resend allowed
		}

		err := code.ReSend()
		assert.NoError(t, err, "should resend even if expired")
		assert.NotEmpty(t, code.code, "code should be regenerated")
		assert.False(t, code.isUsed, "code should not be marked as used after resend")
		assert.WithinDuration(t, time.Now().UTC(), code.updatedAt, 1*time.Second, "updatedAt should be set to now")
		assert.WithinDuration(t, time.Now().UTC().Add(ExpiresAt), code.expiresAt, 1*time.Second, "expiresAt should be set to now + ExpiresAt")
		assert.WithinDuration(
			t,
			time.Now().UTC().Add(ResendTimeout),
			code.resendTimeout,
			1*time.Second,
			"resendTimeout should be set to now + ResendTimeout",
		)
		assert.Greater(t, code.updatedAt, now.Add(-10*time.Minute), "updatedAt should be updated to now")
		assert.Greater(t, code.expiresAt, now.Add(-1*time.Minute), "expiresAt should be updated to now + ExpiresAt")
		assert.Greater(t, code.resendTimeout, now.Add(-1*time.Minute), "resendTimeout should be updated to now + ResendTimeout")
	})
}
