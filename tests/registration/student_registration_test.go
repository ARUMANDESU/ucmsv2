package commands

import (
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
	registrationhttp "github.com/ARUMANDESU/ucms/internal/ports/http/registration"
	"github.com/ARUMANDESU/ucms/tests/integration/builders"
	"github.com/ARUMANDESU/ucms/tests/integration/fixtures"
	"github.com/ARUMANDESU/ucms/tests/integration/framework"
	"github.com/ARUMANDESU/ucms/tests/integration/framework/db"
	"github.com/ARUMANDESU/ucms/tests/integration/framework/event"
	frameworkhttp "github.com/ARUMANDESU/ucms/tests/integration/framework/http"
)

type RegistrationIntegrationSuite struct {
	framework.IntegrationTestSuite
}

func TestRegistrationIntegrationSuite(t *testing.T) {
	suite.Run(t, new(RegistrationIntegrationSuite))
}

func (s *RegistrationIntegrationSuite) TestStudentRegistrationFlow() {
	email := "newstudent@test.com"

	s.DB.SeedGroup(s.T(), fixtures.SEGroup.ID, fixtures.SEGroup.Name, fixtures.SEGroup.Year, fixtures.SEGroup.Major)

	s.T().Run("Start Registration", func(t *testing.T) {
		s.HTTP.StartStudentRegistration(t, email).
			AssertAccepted()
	})

	var reg *db.RegistrationAssertion
	s.T().Run("Verify Registration", func(t *testing.T) {
		reg = s.DB.RequireRegistrationExists(t, email).
			HasStatus(registration.StatusPending).
			HasVerificationCode().
			IsNotExpired()
	})

	var e *registration.RegistrationStarted
	s.T().Run("Verify Registration Event", func(t *testing.T) {
		e = event.RequireEvent(t, s.Event, e)
		require.NotNil(t, e, "Expected RegistrationStarted event to be emitted")
		registration.NewRegistrationStartedAssertion(e).
			AssertRegistrationID(t, reg.GetID()).
			AssertEmail(t, email).
			AssertVerificationCode(t, reg.GetVerificationCode())
	})

	// 4. Verify email sent (wait for async event processing)
	s.T().Run("Verify Email Sent", func(t *testing.T) {
		s.Require().Eventually(func() bool {
			mails := s.MockMailSender.GetSentMails()
			return len(mails) > 0
		}, 5*time.Second, 100*time.Millisecond, "Email should be sent within 5 seconds")

		mails := s.MockMailSender.GetSentMails()
		s.Require().Len(mails, 1)
		s.Equal(email, mails[0].To)
		s.Contains(mails[0].Subject, "Email Verification Code")
		s.Contains(mails[0].Body, reg.GetVerificationCode())
		s.MockMailSender.Reset()
	})

	s.T().Run("Complete Registration", func(t *testing.T) {
		s.HTTP.VerifyRegistrationCode(t, email, reg.GetVerificationCode()).
			AssertSuccess()
	})

	s.T().Run("Complete Student Registration", func(t *testing.T) {
		s.HTTP.CompleteStudentRegistration(t, registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
			Email:            email,
			VerificationCode: reg.GetVerificationCode(),
			Password:         fixtures.TestStudent.Password,
			Barcode:          fixtures.TestStudent.Barcode,
			FirstName:        fixtures.TestStudent.FirstName,
			LastName:         fixtures.TestStudent.LastName,
			GroupId:          fixtures.SEGroup.ID,
		}).AssertSuccess()
	})

	s.T().Run("Verify Student Creation", func(t *testing.T) {
		s.Require().Eventually(func() bool {
			return s.DB.CheckUserExists(t, email)
		}, 5*time.Second, 100*time.Millisecond, "Student should be created within 5 seconds")

		s.DB.RequireUserExists(t, email).
			HasRole(role.Student).
			HasFullName(fixtures.TestStudent.FirstName, fixtures.TestStudent.LastName).
			IsStudent().
			InGroupID(fixtures.SEGroup.ID).
			HasMajor(fixtures.SEGroup.Major.String())
	})

	s.T().Run("Verify Registration Status", func(t *testing.T) {
		s.DB.RequireRegistrationExists(t, email).
			EventuallyHasStatus(registration.StatusCompleted)
	})

	s.T().Run("Verify Welcome Email Sent", func(t *testing.T) {
		s.Require().Eventually(func() bool {
			mails := s.MockMailSender.GetSentMails()
			return len(mails) > 0
		}, 5*time.Second, 100*time.Millisecond, "Welcome email should be sent within 5 seconds")

		mails := s.MockMailSender.GetSentMails()
		s.Require().Len(mails, 1)
		s.Equal(email, mails[0].To)
		s.Contains(mails[0].Subject, "Welcome to UCMS")
		s.Contains(mails[0].Body, fixtures.TestStudent.FirstName)
		s.MockMailSender.Reset()
	})
}

func (s *RegistrationIntegrationSuite) TestStudentRegistrationWithResend() {
	email := "resend@test.com"

	s.T().Run("resend", func(t *testing.T) {
		reg := builders.NewRegistrationBuilder().
			WithEmail(email).
			WithResendAvailable().
			Build()
		s.DB.SeedRegistration(t, reg)

		s.HTTP.ResendVerificationCode(t, email).AssertAccepted()

		e := event.RequireEvent(t, s.Event, &registration.VerificationCodeResent{})
		registration.NewVerificationCodeSentAssertion(e).
			AssertEmail(t, email).
			AssertRegistrationID(t, reg.ID()).
			AssertVerificationCodeNotEqual(t, reg.VerificationCode()).
			AssertVerificationCodeNotEmpty(t)

		s.Require().Eventually(func() bool {
			mails := s.MockMailSender.GetSentMails()
			return len(mails) > 0
		}, 5*time.Second, 100*time.Millisecond, "Email should be sent within 5 seconds")

		mails := s.MockMailSender.GetSentMails()
		s.Require().Len(mails, 1)
		s.Equal(email, mails[0].To)
		s.Contains(mails[0].Subject, "Verification Code Resent")
		s.Contains(mails[0].Body, e.VerificationCode)
		s.MockMailSender.Reset()
	})

	s.T().Run("resend again, should fail", func(t *testing.T) {
		s.HTTP.ResendVerificationCode(t, email).AssertStatus(http.StatusTooManyRequests)
	})
}

func (s *RegistrationIntegrationSuite) TestStudentRegistration_FailPath() {
	s.T().Run("resend timeout is not passed", func(t *testing.T) {
		email := "resend@test.com"
		reg := builders.NewRegistrationBuilder().
			WithEmail(email).
			WithResendNotAvailable().
			Build()
		s.DB.SeedRegistration(s.T(), reg)

		s.HTTP.ResendVerificationCode(t, email).AssertStatus(http.StatusTooManyRequests)
	})

	s.T().Run("registration not exists", func(t *testing.T) {
		email := "notexists@test.com"
		s.HTTP.ResendVerificationCode(t, email).AssertStatus(http.StatusNotFound)
	})
}

func (s *RegistrationIntegrationSuite) TestConcurrentRegistrations() {
	email := "concurrent@test.com"

	// Start multiple registrations concurrently
	var wg sync.WaitGroup
	responses := make([]*frameworkhttp.Response, 3)

	for i := range 3 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			responses[idx] = s.HTTP.StartStudentRegistration(s.T(), email)
		}(i)
	}

	wg.Wait()

	// Only one should succeed
	successCount := 0
	for _, resp := range responses {
		if resp.Code == http.StatusAccepted {
			successCount++
		}
	}

	s.Equal(1, successCount, "Only one registration should succeed")
	s.DB.RequireRegistrationCount(s.T(), 1)

	e := event.RequireEvent(s.T(), s.Event, &registration.RegistrationStarted{})
	registration.NewRegistrationStartedAssertion(e).
		AssertEmail(s.T(), email).
		AssertVerificationCodeNotEmpty(s.T()).
		AssertRegistrationIDNotEmpty(s.T())

	s.Eventually(func() bool {
		mails := s.MockMailSender.GetSentMails()
		return len(mails) > 0
	}, 10*time.Second, 100*time.Millisecond, "Email should be sent within 5 seconds")

	mails := s.MockMailSender.GetSentMails()
	s.Require().Len(mails, 1)
	s.Equal(email, mails[0].To)
	s.Contains(mails[0].Subject, "Email Verification Code")
	s.Contains(mails[0].Body, e.VerificationCode)
}

func (s *RegistrationIntegrationSuite) TestStartRegistrationValidation() {
	s.T().Run("Invalid Email Format", func(t *testing.T) {
		s.HTTP.StartStudentRegistration(t, "invalid-email").
			AssertBadRequest()
	})

	s.T().Run("Empty Email", func(t *testing.T) {
		s.HTTP.StartStudentRegistration(t, "").
			AssertBadRequest()
	})

	s.T().Run("Email Too Long", func(t *testing.T) {
		longEmail := strings.Repeat("a", 250) + "@test.com"
		s.HTTP.StartStudentRegistration(t, longEmail).
			AssertBadRequest()
	})
}

func (s *RegistrationIntegrationSuite) TestVerificationCodeHandling() {
	email := "verify@test.com"
	s.HTTP.StartStudentRegistration(s.T(), email).AssertAccepted()

	s.T().Run("Invalid Verification Code", func(t *testing.T) {
		s.HTTP.VerifyRegistrationCode(t, email, "WRONG1").
			AssertStatus(http.StatusUnprocessableEntity)
	})

	s.T().Run("Too Many Failed Attempts", func(t *testing.T) {
		email := "failed-attempts@test.com"
		s.HTTP.StartStudentRegistration(t, email).AssertAccepted()

		for range registration.MaxVerificationCodeAttempts - 1 {
			s.HTTP.VerifyRegistrationCode(t, email, "WRONG1").
				AssertStatus(http.StatusUnprocessableEntity)
		}
		s.HTTP.VerifyRegistrationCode(t, email, "WRONG1").
			AssertStatus(http.StatusTooManyRequests)

		s.DB.RequireRegistrationExists(t, email).
			HasStatus(registration.StatusExpired).
			HasCodeAttempts(3)
	})

	s.T().Run("Verify Already Expired Code", func(t *testing.T) {
		email := "expired@test.com"
		expiredReg := s.Builder.Registration.ExpiredRegistration(email)
		s.DB.SeedRegistration(s.T(), expiredReg)

		reg := s.DB.RequireRegistrationExists(t, email)
		s.HTTP.VerifyRegistrationCode(t, email, reg.GetVerificationCode()).
			AssertStatus(http.StatusUnprocessableEntity)
	})
}

func (s *RegistrationIntegrationSuite) TestCompleteRegistrationValidation() {
	s.T().Run("Weak Password", func(t *testing.T) {
		email := "weak-password@test.com"
		s.setupVerifiedRegistration(email)

		s.HTTP.CompleteStudentRegistration(s.T(), registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
			Email:            email,
			VerificationCode: s.getVerificationCode(email),
			Password:         "weak",
			Barcode:          "STU001",
			FirstName:        "Test",
			LastName:         "Student",
			GroupId:          fixtures.SEGroup.ID,
		}).AssertBadRequest()
	})

	s.T().Run("Duplicate Barcode", func(t *testing.T) {
		email := "duplicate@test.com"
		s.setupVerifiedRegistration(email)
		student := s.Builder.User.Student("existing@test.com")

		s.DB.SeedUser(s.T(), student.User())
		s.DB.SeedStudent(s.T(), student.User().Barcode(), fixtures.SEGroup.ID)

		s.HTTP.CompleteStudentRegistration(s.T(), registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
			Email:            email,
			VerificationCode: s.getVerificationCode(email),
			Password:         fixtures.TestStudent.Password,
			Barcode:          string(student.User().Barcode()),
			FirstName:        "Test",
			LastName:         "Student",
			GroupId:          fixtures.SEGroup.ID,
		}).AssertStatus(http.StatusConflict)
	})

	s.T().Run("Invalid Group ID", func(t *testing.T) {
		email := "invalid-group@test.com"
		s.setupVerifiedRegistration(email)
		invalidGroupID := uuid.New()

		s.HTTP.CompleteStudentRegistration(s.T(), registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
			Email:            email,
			VerificationCode: s.getVerificationCode(email),
			Password:         fixtures.TestStudent.Password,
			Barcode:          "STU002",
			FirstName:        "Test",
			LastName:         "Student",
			GroupId:          invalidGroupID,
		}).AssertStatus(http.StatusNotFound)
	})
}

func (s *RegistrationIntegrationSuite) TestRegistrationStates() {
	s.T().Run("Complete Without Verification", func(t *testing.T) {
		email := "no-verify@test.com"
		s.DB.SeedGroup(s.T(), fixtures.SEGroup.ID, fixtures.SEGroup.Name, fixtures.SEGroup.Year, fixtures.SEGroup.Major)

		s.HTTP.StartStudentRegistration(t, email).AssertAccepted()
		reg := s.DB.RequireRegistrationExists(t, email)

		s.HTTP.CompleteStudentRegistration(t, registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
			Email:            email,
			VerificationCode: reg.GetVerificationCode(),
			Password:         fixtures.TestStudent.Password,
			Barcode:          "STU003",
			FirstName:        "Test",
			LastName:         "Student",
			GroupId:          fixtures.SEGroup.ID,
		}).AssertBadRequest()
	})

	s.T().Run("Double Complete", func(t *testing.T) {
		email := "double-complete@test.com"
		s.setupCompletedRegistration(email)

		s.HTTP.CompleteStudentRegistration(s.T(), registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
			Email:            email,
			VerificationCode: s.getVerificationCode(email),
			Password:         fixtures.TestStudent.Password,
			Barcode:          "STU004",
			FirstName:        "Test",
			LastName:         "Student",
			GroupId:          fixtures.SEGroup.ID,
		}).AssertStatus(http.StatusConflict)
	})
}

func (s *RegistrationIntegrationSuite) TestBusinessRules() {
	s.T().Run("Registration Already Exists", func(t *testing.T) {
		email := "existing@test.com"
		s.HTTP.StartStudentRegistration(t, email).AssertAccepted()
		s.HTTP.StartStudentRegistration(t, email).AssertStatus(http.StatusTooManyRequests)
	})

	s.T().Run("Name Length Validation", func(t *testing.T) {
		email := "names@test.com"
		s.setupVerifiedRegistration(email)

		s.HTTP.CompleteStudentRegistration(s.T(), registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
			Email:            email,
			VerificationCode: s.getVerificationCode(email),
			Password:         fixtures.TestStudent.Password,
			Barcode:          "STU005",
			FirstName:        "X",
			LastName:         strings.Repeat("A", 101),
			GroupId:          fixtures.SEGroup.ID,
		}).AssertBadRequest()
	})
}

func (s *RegistrationIntegrationSuite) TestRegistration_StudentComplete_RequestValidation() {
	s.DB.SeedGroup(s.T(), fixtures.SEGroup.ID, fixtures.SEGroup.Name, fixtures.SEGroup.Year, fixtures.SEGroup.Major)

	tests := []struct {
		name           string
		setup          func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody)
		expectedStatus int
		message        string
		setupBefore    bool
	}{
		{
			name: "Empty Email",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Email = ""
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Email Address cannot be blank",
		},
		{
			name: "Invalid Email Format",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Email = "invalid-email"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Email Address must be a valid email address",
		},
		{
			name: "Empty Verification Code",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.VerificationCode = ""
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Verification Code cannot be blank",
		},
		{
			name: "Empty Password",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Password = ""
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Password cannot be blank",
		},
		{
			name: "Empty Barcode",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Barcode = ""
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Barcode cannot be blank",
		},
		{
			name: "Invalid Barcode Format",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Barcode = "INVALID-BARCODE"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Barcode must contain English letters and digits only",
		},
		{
			name: "Empty First Name",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.FirstName = ""
			},
			expectedStatus: http.StatusBadRequest,
			message:        "First Name cannot be blank",
		},
		{
			name: "Empty Last Name",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.LastName = ""
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Last Name cannot be blank",
		},
		{
			name: "Invalid Group ID",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.GroupId = uuid.New()
			},
			expectedStatus: http.StatusNotFound,
			message:        "Academic Group not found",
		},
		{
			name: "Group ID Not Provided",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.GroupId = uuid.Nil
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Group ID Not Found",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.GroupId = uuid.New()
			},
			expectedStatus: http.StatusNotFound,
			message:        "Academic Group not found",
		},
		// Password validation test cases
		{
			name: "Password Too Short",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Password = "Pass1!"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Password the length must be between 8 and 128",
		},
		{
			name: "Password Too Long",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Password = strings.Repeat("Password1!", 15) // 150 characters
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Password the length must be between 8 and 128",
		},
		{
			name: "Password Missing Uppercase",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Password = "password123!"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Password must contain at least 8 characters with uppercase, lowercase, number, and special character",
		},
		{
			name: "Password Missing Lowercase",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Password = "PASSWORD123!"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Password must contain at least 8 characters with uppercase, lowercase, number, and special character",
		},
		{
			name: "Password Missing Number",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Password = "Password!"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Password must contain at least 8 characters with uppercase, lowercase, number, and special character",
		},
		{
			name: "Password Missing Special Character",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Password = "Password123"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Password must contain at least 8 characters with uppercase, lowercase, number, and special character",
		},
		// Name validation test cases
		{
			name: "First Name Too Long",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.FirstName = strings.Repeat("A", 151)
			},
			expectedStatus: http.StatusBadRequest,
			message:        "First Name the length must be between 1 and 150",
		},
		{
			name: "Last Name Too Long",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.LastName = strings.Repeat("B", 151)
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Last Name the length must be between 1 and 150",
		},
		{
			name: "First Name Invalid Characters",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.FirstName = "John123"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "must be a valid name",
		},
		{
			name: "Last Name Invalid Characters",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.LastName = "Smith@#$"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "must be a valid name",
		},
		// valid names with special characters
		{
			name: "First Name With Valid Special Characters",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Barcode = "STU001"
				req.Email = "valid-firstname-1@test.com"
				req.FirstName = "Jean-Pierre"
			},
			expectedStatus: http.StatusOK,
			setupBefore:    true,
		},
		{
			name: "Last Name With Apostrophe",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Barcode = "STU002"
				req.Email = "valid-lastname-1@test.com"
				req.LastName = "O'Connor"
			},
			expectedStatus: http.StatusOK,
			setupBefore:    true,
		},
		// Email validation edge cases
		{
			name: "Email Too Long",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				longEmail := strings.Repeat("a", 250) + "@test.com"
				req.Email = longEmail
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Email Address must be a valid email address",
		},
		{
			name: "Email Too Short",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Email = "a@b"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Email Address must be a valid email address",
		},
		// Barcode validation edge cases
		{
			name: "Barcode Too Long",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Barcode = strings.Repeat("A", 21) // Over typical barcode length
				req.Email = "barcode-too-long@test.com"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Barcode the length must be between 6 and 20",
			setupBefore:    true,
		},
		{
			name: "Barcode With Spaces",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Barcode = "STU 001"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Barcode must contain English letters and digits only",
		},
		{
			name: "Barcode With Unicode Characters",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Barcode = "STUদেন্ট001"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Barcode must contain English letters and digits only",
		},
		// Verification code edge cases
		{
			name: "Verification Code Too Long",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.VerificationCode = strings.Repeat("1", 20)
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Verification Code",
		},
		{
			name: "Verification Code With Special Characters",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.VerificationCode = "123@#$"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Verification Code",
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			request := registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
				Email:            fixtures.TestStudent.Email,
				VerificationCode: "123456",
				Password:         fixtures.TestStudent.Password,
				Barcode:          fixtures.TestStudent.Barcode,
				FirstName:        fixtures.TestStudent.FirstName,
				LastName:         fixtures.TestStudent.LastName,
				GroupId:          fixtures.SEGroup.ID,
			}
			tt.setup(&request)
			if tt.setupBefore {
				s.setupVerifiedRegistration(request.Email)
				request.VerificationCode = s.getVerificationCode(request.Email)
			}
			response := s.HTTP.CompleteStudentRegistration(t, request)
			response.AssertStatus(tt.expectedStatus).AssertContainsMessage(tt.message)
		})
	}
}

func (s *RegistrationIntegrationSuite) TestRegistration_StudentComplete_BusinessErrors() {
	s.DB.SeedGroup(s.T(), fixtures.SEGroup.ID, fixtures.SEGroup.Name, fixtures.SEGroup.Year, fixtures.SEGroup.Major)

	tests := []struct {
		name            string
		setup           func(t *testing.T) (email, verificationCode, barcode string)
		expectedStatus  int
		expectedMessage string
	}{
		{
			name: "Registration Not Found",
			setup: func(t *testing.T) (string, string, string) {
				return "nonexistent@test.com", "123456", "STU001"
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "Resource not found",
		},
		{
			name: "Email Not Verified First",
			setup: func(t *testing.T) (string, string, string) {
				email := "not-verified@test.com"
				s.HTTP.StartStudentRegistration(t, email).AssertAccepted()
				reg := s.DB.RequireRegistrationExists(t, email)
				return email, reg.GetVerificationCode(), "STU002"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Please verify your email first",
		},
		{
			name: "Invalid Verification Code",
			setup: func(t *testing.T) (string, string, string) {
				email := "invalid-code@test.com"
				s.HTTP.StartStudentRegistration(t, email).AssertAccepted()
				s.DB.RequireRegistrationExists(t, email)
				return email, "WRONG1", "STU003"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Please verify your email first", // Since email not verified, it should fail with verify first
		},
		{
			name: "Invalid Verification Code Length",
			setup: func(t *testing.T) (string, string, string) {
				email := "invalid-code-length@test.com"
				s.setupVerifiedRegistration(email)
				return email, "WRONG123", "STU004"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Verification Code the length must be exactly 6",
		},
		{
			name: "Registration Already Completed",
			setup: func(t *testing.T) (string, string, string) {
				email := "completed@test.com"
				s.setupCompletedRegistration(email)
				return email, s.getVerificationCode(email), "STU005"
			},
			expectedStatus:  http.StatusConflict,
			expectedMessage: "This email address is already registered",
		},
		{
			name: "Duplicate Student Barcode",
			setup: func(t *testing.T) (string, string, string) {
				email := "duplicate-barcode@test.com"
				s.setupVerifiedRegistration(email)

				// Create an existing student with the same barcode
				existingStudent := s.Builder.User.Student("existing@test.com")
				s.DB.SeedUser(t, existingStudent.User())
				s.DB.SeedStudent(t, existingStudent.User().Barcode(), fixtures.SEGroup.ID)

				return email, s.getVerificationCode(email), existingStudent.User().Barcode().String()
			},
			expectedStatus:  http.StatusConflict,
			expectedMessage: "This barcode is already in use",
		},
		{
			name: "User Already Exists With Email",
			setup: func(t *testing.T) (string, string, string) {
				email := "existing-user@test.com"
				s.setupVerifiedRegistration(email)

				// Create an existing user with the same email
				existingUser := builders.NewStudentBuilder().WithEmail(email).WithBarcode("STU006").Build()
				s.DB.SeedUser(t, existingUser.User())
				s.DB.SeedStudent(t, existingUser.User().Barcode(), fixtures.SEGroup.ID)

				return email, s.getVerificationCode(email), "STU007"
			},
			expectedStatus:  http.StatusConflict,
			expectedMessage: "This email address is already registered",
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			email, verificationCode, barcode := tt.setup(t)

			response := s.HTTP.CompleteStudentRegistration(t, registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
				Email:            email,
				VerificationCode: verificationCode,
				Password:         fixtures.TestStudent.Password,
				Barcode:          barcode,
				FirstName:        fixtures.TestStudent.FirstName,
				LastName:         fixtures.TestStudent.LastName,
				GroupId:          fixtures.SEGroup.ID,
			})

			response.AssertStatus(tt.expectedStatus)
			if tt.expectedMessage != "" {
				response.AssertContainsMessage(tt.expectedMessage)
			}
		})
	}
}

func (s *RegistrationIntegrationSuite) TestRegistration_StudentComplete_VerificationCodeExpired() {
	s.T().Run("Expired Verification Code", func(t *testing.T) {
		email := "expired-code@test.com"
		s.DB.SeedGroup(t, fixtures.SEGroup.ID, fixtures.SEGroup.Name, fixtures.SEGroup.Year, fixtures.SEGroup.Major)

		// Create an expired registration
		expiredReg := s.Builder.Registration.ExpiredRegistration(email)
		s.DB.SeedRegistration(t, expiredReg)

		response := s.HTTP.CompleteStudentRegistration(t, registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
			Email:            email,
			VerificationCode: expiredReg.VerificationCode(),
			Password:         fixtures.TestStudent.Password,
			Barcode:          "EXPSTU001",
			FirstName:        fixtures.TestStudent.FirstName,
			LastName:         fixtures.TestStudent.LastName,
			GroupId:          fixtures.SEGroup.ID,
		})

		response.AssertStatus(http.StatusBadRequest).
			AssertContainsMessage("Please verify your email first")
	})
}

// TestRegistration_StudentComplete_SecurityValidation tests security-related validation
func (s *RegistrationIntegrationSuite) TestRegistration_StudentComplete_SecurityValidation() {
	s.DB.SeedGroup(s.T(), fixtures.SEGroup.ID, fixtures.SEGroup.Name, fixtures.SEGroup.Year, fixtures.SEGroup.Major)

	tests := []struct {
		name    string
		setup   func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody)
		message string
	}{
		{
			name: "SQL Injection in Email",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Email = "test'; DROP TABLE users; --@test.com"
			},
			message: "Email Address must be a valid email address",
		},
		{
			name: "XSS in First Name",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.FirstName = "<script>alert('xss')</script>"
			},
			message: "must be a valid name",
		},
		{
			name: "XSS in Last Name",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.LastName = "<img src=x onerror=alert('xss')>"
			},
			message: "must be a valid name",
		},
		{
			name: "HTML Entities in Name",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.FirstName = "&lt;script&gt;alert('test')&lt;/script&gt;"
			},
			message: "must be a valid name",
		},
		{
			name: "Null Bytes in Barcode",
			setup: func(req *registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody) {
				req.Barcode = "STU001\x00admin"
			},
			message: "Barcode must contain English letters and digits only",
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			request := registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
				Email:            fixtures.TestStudent.Email,
				VerificationCode: "123456",
				Password:         fixtures.TestStudent.Password,
				Barcode:          fixtures.TestStudent.Barcode,
				FirstName:        fixtures.TestStudent.FirstName,
				LastName:         fixtures.TestStudent.LastName,
				GroupId:          fixtures.SEGroup.ID,
			}
			tt.setup(&request)

			response := s.HTTP.CompleteStudentRegistration(t, request)
			response.AssertBadRequest().AssertContainsMessage(tt.message)
		})
	}
}

func (s *RegistrationIntegrationSuite) setupVerifiedRegistration(email string) {
	if !s.DB.CheckGroupExists(s.T(), fixtures.SEGroup.ID) {
		s.DB.SeedGroup(s.T(), fixtures.SEGroup.ID, fixtures.SEGroup.Name, fixtures.SEGroup.Year, fixtures.SEGroup.Major)
	}
	s.HTTP.StartStudentRegistration(s.T(), email).AssertAccepted()
	reg := s.DB.RequireRegistrationExists(s.T(), email)
	s.HTTP.VerifyRegistrationCode(s.T(), email, reg.GetVerificationCode()).AssertSuccess()
}

func (s *RegistrationIntegrationSuite) setupCompletedRegistration(email string) {
	s.setupVerifiedRegistration(email)
	s.HTTP.CompleteStudentRegistration(s.T(), registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
		Email:            email,
		VerificationCode: s.getVerificationCode(email),
		Password:         fixtures.TestStudent.Password,
		Barcode:          "STU999",
		FirstName:        "Test",
		LastName:         "Student",
		GroupId:          fixtures.SEGroup.ID,
	}).AssertSuccess()
}

func (s *RegistrationIntegrationSuite) setupCompletedRegistrationWith(email, barcode string, groupID uuid.UUID) {
	s.setupVerifiedRegistration(email)
	s.HTTP.CompleteStudentRegistration(s.T(), registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
		Email:            email,
		VerificationCode: s.getVerificationCode(email),
		Password:         fixtures.TestStudent.Password,
		Barcode:          barcode,
		FirstName:        fixtures.TestStudent.FirstName,
		LastName:         fixtures.TestStudent.LastName,
		GroupId:          groupID,
	}).AssertSuccess()
}

func (s *RegistrationIntegrationSuite) getVerificationCode(email string) string {
	return s.DB.RequireRegistrationExists(s.T(), email).GetVerificationCode()
}
