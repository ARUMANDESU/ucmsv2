package commands

import (
	"net/http"
	"sync"
	"testing"
	"time"

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

	var e *event.RegistrationStartedAssertion
	s.T().Run("Verify Registration Event", func(t *testing.T) {
		e = s.Event.AssertRegistrationStartedEvent(t, email).
			HasEmail(email).
			HasVerificationCode().
			HasRegistrationID(reg.GetID())
	})

	// 4. Verify email sent (wait for async event processing)
	s.T().Run("Verify Email Sent", func(t *testing.T) {
		s.Event.AssertRegistrationStartedEvent(t, email)

		s.Require().Eventually(func() bool {
			mails := s.MockMailSender.GetSentMails()
			return len(mails) > 0
		}, 5*time.Second, 100*time.Millisecond, "Email should be sent within 5 seconds")

		mails := s.MockMailSender.GetSentMails()
		s.Require().Len(mails, 1)
		s.Equal(email, mails[0].To)
		s.Contains(mails[0].Subject, "Email Verification Code")
		s.Contains(mails[0].Body, e.GetVerificationCode())
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

	event := s.Event.AssertRegistrationStartedEvent(s.T(), email).
		HasEmail(email).
		HasVerificationCode()

	s.Eventually(func() bool {
		mails := s.MockMailSender.GetSentMails()
		return len(mails) > 0
	}, 10*time.Second, 100*time.Millisecond, "Email should be sent within 5 seconds")

	mails := s.MockMailSender.GetSentMails()
	s.Require().Len(mails, 1)
	s.Equal(email, mails[0].To)
	s.Contains(mails[0].Subject, "Email Verification Code")
	s.Contains(mails[0].Body, event.GetVerificationCode())
}
