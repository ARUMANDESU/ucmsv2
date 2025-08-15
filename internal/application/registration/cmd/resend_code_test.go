package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/tests/integration/builders"
	"github.com/ARUMANDESU/ucms/tests/mocks"
)

type ResendCodeSuite struct {
	Handler      *ResendCodeHandler
	MockRepo     *mocks.RegistrationRepo
	MockUserRepo *mocks.UserRepo
}

func NewResendCodeSuite(t *testing.T) *ResendCodeSuite {
	t.Helper()

	mockRepo := mocks.NewRegistrationRepo()
	mockUserRepo := mocks.NewUserRepo()

	handler := NewResendCodeHandler(ResendCodeHandlerArgs{
		Repo:       mockRepo,
		UserGetter: mockUserRepo,
	})

	return &ResendCodeSuite{
		Handler:      handler,
		MockRepo:     mockRepo,
		MockUserRepo: mockUserRepo,
	}
}

func TestResendCodeHandler_HappyPath(t *testing.T) {
	t.Parallel()

	t.Run("pending and resend available registration", func(t *testing.T) {
		s := NewResendCodeSuite(t)
		email := "happypath1@test.com"
		reg := builders.NewRegistrationBuilder().
			WithEmail(email).
			WithResendAvailable().
			Build()
		originalCode := reg.VerificationCode()
		s.MockRepo.SeedRegistration(t, reg)

		cmd := ResendCode{
			Email: reg.Email(),
		}

		err := s.Handler.Handle(t.Context(), cmd)
		require.NoError(t, err)

		s.MockRepo.AssertRegistrationExistsByEmail(t, reg.Email()).
			AssertStatus(t, registration.StatusPending).
			AssertCodeAttempts(t, 0).
			AssertResendNotAvailable(t).
			AssertVerificationCodeIsNot(t, originalCode).
			AssertVerificationCodeNotEmpty(t)

		e := mocks.RequireEventExists(t, s.MockRepo.EventRepo, &registration.VerificationCodeResent{})
		registration.NewVerificationCodeSentAssertion(e).
			AssertRegistrationID(t, reg.ID()).
			AssertEmail(t, reg.Email()).
			AssertVerificationCode(t, reg.VerificationCode())
	})

	t.Run("expired and resend available registration", func(t *testing.T) {
		s := NewResendCodeSuite(t)
		email := "happypath2@test.com"
		reg := builders.NewRegistrationBuilder().
			WithEmail(email).
			WithExpiredCode().
			WithResendAvailable().
			Build()
		originalCode := reg.VerificationCode()
		s.MockRepo.SeedRegistration(t, reg)

		cmd := ResendCode{
			Email: reg.Email(),
		}

		err := s.Handler.Handle(t.Context(), cmd)
		require.NoError(t, err)

		s.MockRepo.AssertRegistrationExistsByEmail(t, reg.Email()).
			AssertStatus(t, registration.StatusPending).
			AssertCodeAttempts(t, 0).
			AssertResendNotAvailable(t).
			AssertVerificationCodeIsNot(t, originalCode).
			AssertVerificationCodeNotEmpty(t)

		e := mocks.RequireEventExists(t, s.MockRepo.EventRepo, &registration.VerificationCodeResent{})
		registration.NewVerificationCodeSentAssertion(e).
			AssertRegistrationID(t, reg.ID()).
			AssertEmail(t, reg.Email()).
			AssertVerificationCode(t, reg.VerificationCode())
	})

	t.Run("max attempts reached", func(t *testing.T) {
		s := NewResendCodeSuite(t)
		email := "maxattempts@test.com"
		reg := builders.NewRegistrationBuilder().
			WithEmail(email).
			WithResendAvailable().
			WithMaxAttemptsReached().
			Build()
		originalCode := reg.VerificationCode()
		s.MockRepo.SeedRegistration(t, reg)

		cmd := ResendCode{
			Email: email,
		}

		err := s.Handler.Handle(t.Context(), cmd)
		require.NoError(t, err)

		s.MockRepo.AssertRegistrationExistsByEmail(t, reg.Email()).
			AssertStatus(t, registration.StatusPending).
			AssertCodeAttempts(t, 0).
			AssertResendNotAvailable(t).
			AssertVerificationCodeIsNot(t, originalCode).
			AssertVerificationCodeNotEmpty(t)

		e := mocks.RequireEventExists(t, s.MockRepo.EventRepo, &registration.VerificationCodeResent{})
		registration.NewVerificationCodeSentAssertion(e).
			AssertRegistrationID(t, reg.ID()).
			AssertEmail(t, reg.Email()).
			AssertVerificationCode(t, reg.VerificationCode())
	})
}

func TestResendCodeHandler_ErrorCases(t *testing.T) {
	t.Parallel()

	s := NewResendCodeSuite(t)

	t.Run("empty email", func(t *testing.T) {
		cmd := ResendCode{
			Email: "",
		}

		err := s.Handler.Handle(t.Context(), cmd)
		require.Error(t, err)
		assert.ErrorIs(t, err, user.ErrMissingEmail)
	})

	t.Run("user already exists", func(t *testing.T) {
		email := "existing@test.com"
		existingUser := builders.NewUserBuilder().
			WithEmail(email).
			Build()
		s.MockUserRepo.SeedUser(t, existingUser)

		cmd := ResendCode{
			Email: email,
		}

		err := s.Handler.Handle(t.Context(), cmd)
		require.Error(t, err)
		assert.True(t, errorx.IsDuplicateEntry(err))
	})

	t.Run("registration not found", func(t *testing.T) {
		email := "nonexistent@test.com"

		cmd := ResendCode{
			Email: email,
		}

		err := s.Handler.Handle(t.Context(), cmd)
		require.Error(t, err)
		assert.True(t, errorx.IsNotFound(err), "expected NotFound error, got: %v", err)
	})

	t.Run("resend not available - timeout not reached", func(t *testing.T) {
		email := "timeout@test.com"
		reg := builders.NewRegistrationBuilder().
			WithEmail(email).
			WithResendNotAvailable().
			Build()
		s.MockRepo.SeedRegistration(t, reg)

		cmd := ResendCode{
			Email: email,
		}

		err := s.Handler.Handle(t.Context(), cmd)
		require.Error(t, err)
		assert.ErrorIs(t, err, registration.ErrWaitUntilResend)
	})

	t.Run("registration completed", func(t *testing.T) {
		email := "completed@test.com"
		reg := builders.NewRegistrationBuilder().
			WithEmail(email).
			WithResendAvailable().
			WithStatus(registration.StatusCompleted).
			Build()
		s.MockRepo.SeedRegistration(t, reg)

		cmd := ResendCode{
			Email: email,
		}

		err := s.Handler.Handle(t.Context(), cmd)
		require.Error(t, err)
		assert.ErrorIs(t, err, registration.ErrRegistrationCompleted)
	})
}

func TestResendCodeHandler_EdgeCases(t *testing.T) {
	t.Parallel()

	s := NewResendCodeSuite(t)

	t.Run("whitespace only email", func(t *testing.T) {
		cmd := ResendCode{
			Email: "   ",
		}

		err := s.Handler.Handle(t.Context(), cmd)
		require.Error(t, err)
	})

	t.Run("registration with pending status but code not expired", func(t *testing.T) {
		email := "validcode@test.com"
		reg := builders.NewRegistrationBuilder().
			WithEmail(email).
			WithStatus(registration.StatusPending).
			WithResendAvailable().
			Build()
		originalCode := reg.VerificationCode()
		s.MockRepo.SeedRegistration(t, reg)

		cmd := ResendCode{
			Email: email,
		}

		err := s.Handler.Handle(t.Context(), cmd)
		require.NoError(t, err)

		s.MockRepo.AssertRegistrationExistsByEmail(t, email).
			AssertStatus(t, registration.StatusPending).
			AssertVerificationCodeIsNot(t, originalCode).
			AssertVerificationCodeNotEmpty(t)
	})
}
