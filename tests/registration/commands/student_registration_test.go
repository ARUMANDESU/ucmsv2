package commands

import (
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
	registrationhttp "github.com/ARUMANDESU/ucms/internal/ports/http/registration"
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
	})

	s.T().Run("Complete Registration", func(t *testing.T) {
		s.HTTP.VerifyRegistrationCode(t, email, reg.GetVerificationCode()).
			AssertSuccess()
	})

	s.T().Run("Complete Student Registration", func(t *testing.T) {
		s.HTTP.CompleteStudentRegistration(t, registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
			Email:            openapi_types.Email(email),
			VerificationCode: reg.GetVerificationCode(),
			Password:         fixtures.TestStudent.Password,
			Barcode:          fixtures.TestStudent.ID,
			FirstName:        fixtures.TestStudent.FirstName,
			LastName:         fixtures.TestStudent.LastName,
			GroupId:          registrationhttp.GroupID(fixtures.SEGroup.ID),
		}).AssertSuccess()
	})

	s.T().Run("Verify Student Registration Completed Event", func(t *testing.T) {
		e := event.RequireEvent(t, s.Event, &registration.StudentRegistrationCompleted{})
		require.NotNil(t, e, "Expected StudentRegistered event to be emitted")
		assert.Equal(t, reg.GetID(), e.RegistrationID)
		assert.Equal(t, fixtures.TestStudent.ID, e.Barcode)
		assert.Equal(t, email, e.Email)
		assert.Equal(t, fixtures.TestStudent.FirstName, e.FirstName)
		assert.Equal(t, fixtures.TestStudent.LastName, e.LastName)
		assert.Equal(t, fixtures.SEGroup.ID.String(), e.GroupID.String())
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(e.PassHash), []byte(fixtures.TestStudent.Password)))
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
			HasStatus(registration.StatusCompleted)
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
			Email:            openapi_types.Email(email),
			VerificationCode: s.getVerificationCode(email),
			Password:         "weak",
			Barcode:          "STU001",
			FirstName:        "Test",
			LastName:         "Student",
			GroupId:          registrationhttp.GroupID(fixtures.SEGroup.ID),
		}).AssertBadRequest()
	})

	s.T().Run("Duplicate Barcode", func(t *testing.T) {
		email := "duplicate@test.com"
		s.setupVerifiedRegistration(email)
		student := s.Builder.User.Student("existing@test.com")

		s.DB.SeedUser(s.T(), student.User())
		s.DB.SeedStudent(s.T(), student.User().ID(), fixtures.SEGroup.ID)

		s.HTTP.CompleteStudentRegistration(s.T(), registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
			Email:            openapi_types.Email(email),
			VerificationCode: s.getVerificationCode(email),
			Password:         fixtures.TestStudent.Password,
			Barcode:          string(student.User().ID()),
			FirstName:        "Test",
			LastName:         "Student",
			GroupId:          registrationhttp.GroupID(fixtures.SEGroup.ID),
		}).AssertStatus(http.StatusConflict)
	})

	s.T().Run("Invalid Group ID", func(t *testing.T) {
		email := "invalid-group@test.com"
		s.setupVerifiedRegistration(email)
		invalidGroupID := uuid.New()

		s.HTTP.CompleteStudentRegistration(s.T(), registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
			Email:            openapi_types.Email(email),
			VerificationCode: s.getVerificationCode(email),
			Password:         fixtures.TestStudent.Password,
			Barcode:          "STU002",
			FirstName:        "Test",
			LastName:         "Student",
			GroupId:          registrationhttp.GroupID(invalidGroupID),
		}).AssertStatus(http.StatusNotFound)
	})
}

//	func (s *RegistrationIntegrationSuite) TestRegistrationStates() {
//		s.T().Run("Complete Without Verification", func(t *testing.T) {
//			email := "no-verify@test.com"
//			s.DB.SeedGroup(s.T(), fixtures.SEGroup.ID, fixtures.SEGroup.Name, fixtures.SEGroup.Year, fixtures.SEGroup.Major)
//
//			s.HTTP.StartStudentRegistration(t, email).AssertAccepted()
//			reg := s.DB.RequireRegistrationExists(t, email)
//
//			s.HTTP.CompleteStudentRegistration(t, registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
//				Email:            openapi_types.Email(email),
//				VerificationCode: reg.GetVerificationCode(),
//				Password:         fixtures.TestStudent.Password,
//				Barcode:          "STU003",
//				FirstName:        "Test",
//				LastName:         "Student",
//				GroupId:          registrationhttp.GroupID(fixtures.SEGroup.ID),
//			}).AssertSuccess()
//		})
//
//		s.T().Run("Double Complete", func(t *testing.T) {
//			email := "double-complete@test.com"
//			s.setupCompletedRegistration(email)
//
//			s.HTTP.CompleteStudentRegistration(s.T(), registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
//				Email:            openapi_types.Email(email),
//				VerificationCode: s.getVerificationCode(email),
//				Password:         fixtures.TestStudent.Password,
//				Barcode:          "STU004",
//				FirstName:        "Test",
//				LastName:         "Student",
//				GroupId:          registrationhttp.GroupID(fixtures.SEGroup.ID),
//			}).AssertStatus(http.StatusUnprocessableEntity)
//		})
//	}
//
//	func (s *RegistrationIntegrationSuite) TestBusinessRules() {
//		s.T().Run("Registration Already Exists", func(t *testing.T) {
//			email := "existing@test.com"
//			s.HTTP.StartStudentRegistration(t, email).AssertAccepted()
//			s.HTTP.StartStudentRegistration(t, email).AssertStatus(http.StatusConflict)
//		})
//
//		s.T().Run("Name Length Validation", func(t *testing.T) {
//			email := "names@test.com"
//			s.setupVerifiedRegistration(email)
//
//			s.HTTP.CompleteStudentRegistration(s.T(), registrationhttp.PostV1RegistrationsStudentsCompleteJSONRequestBody{
//				Email:            openapi_types.Email(email),
//				VerificationCode: s.getVerificationCode(email),
//				Password:         fixtures.TestStudent.Password,
//				Barcode:          "STU005",
//				FirstName:        "X",
//				LastName:         strings.Repeat("A", 101),
//				GroupId:          registrationhttp.GroupID(fixtures.SEGroup.ID),
//			}).AssertBadRequest()
//		})
//	}
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
		Email:            openapi_types.Email(email),
		VerificationCode: s.getVerificationCode(email),
		Password:         fixtures.TestStudent.Password,
		Barcode:          "STU999",
		FirstName:        "Test",
		LastName:         "Student",
		GroupId:          registrationhttp.GroupID(fixtures.SEGroup.ID),
	}).AssertSuccess()
}

func (s *RegistrationIntegrationSuite) getVerificationCode(email string) string {
	return s.DB.RequireRegistrationExists(s.T(), email).GetVerificationCode()
}
