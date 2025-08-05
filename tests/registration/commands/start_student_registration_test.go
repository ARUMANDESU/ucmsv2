package commands

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/tests/integration/framework"
	frameworkhttp "github.com/ARUMANDESU/ucms/tests/integration/framework/http"
)

type RegistrationIntegrationSuite struct {
	framework.IntegrationTestSuite
}

func TestRegistrationIntegrationSuite(t *testing.T) {
	suite.Run(t, new(RegistrationIntegrationSuite))
}

func (s *RegistrationIntegrationSuite) TestStudentRegistrationFlow() {
	// Test complete registration flow
	email := "newstudent@test.com"

	// 1. Start registration
	s.HTTP.StartStudentRegistration(s.T(), email).
		AssertAccepted()
	// 2. Verify registration created
	reg := s.DB.AssertRegistrationExists(s.T(), email).
		HasStatus(registration.StatusPending).
		HasVerificationCode().
		IsNotExpired()

	// 3. Verify event published
	event := s.Event.AssertRegistrationStartedEvent(s.T(), email).
		HasEmail(email).
		HasVerificationCode().
		HasRegistrationID(reg.GetID())

	// 4. Verify email sent (wait for async event processing)
	s.Eventually(func() bool {
		mails := s.MockMailSender.GetSentMails()
		return len(mails) > 0
	}, 5*time.Second, 100*time.Millisecond, "Email should be sent within 5 seconds")

	mails := s.MockMailSender.GetSentMails()
	s.Require().Len(mails, 1)
	s.Equal(email, mails[0].To)
	s.Contains(mails[0].Subject, "Email Verification Code")
	s.Contains(mails[0].Body, event.GetVerificationCode())

	// // 5. Verify code
	// s.HTTP.VerifyRegistrationCode(s.T(), email, reg.GetVerificationCode()).
	// 	AssertSuccess()
	//
	// // 6. Complete registration
	// s.HTTP.CompleteStudentRegistration(s.T(), frameworkhttp.CompleteRegistrationRequest{
	// 	Email:     email,
	// 	FirstName: "John",
	// 	LastName:  "Doe",
	// 	Password:  "SecurePass123!",
	// 	GroupID:   fixtures.SEGroup.ID.String(),
	// }).AssertSuccess()
	//
	// // 7. Verify user created
	// s.DB.AssertUserExists(s.T(), email).
	// 	HasRole("student").
	// 	HasFullName("John", "Doe").
	// 	IsStudent().
	// 	InGroup("SE-2301").
	// 	HasMajor("Software Engineering")
	//
	// // 8. Verify registration completed
	// s.DB.AssertRegistrationExists(s.T(), email).
	// 	HasStatus(registration.StatusCompleted)
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
	s.DB.AssertRegistrationCount(s.T(), 1)

	event := s.Event.AssertRegistrationStartedEvent(s.T(), email).
		HasEmail(email).
		HasVerificationCode()

	s.Eventually(func() bool {
		mails := s.MockMailSender.GetSentMails()
		return len(mails) > 0
	}, 5*time.Second, 100*time.Millisecond, "Email should be sent within 5 seconds")

	mails := s.MockMailSender.GetSentMails()
	s.Require().Len(mails, 1)
	s.Equal(email, mails[0].To)
	s.Contains(mails[0].Subject, "Email Verification Code")
	s.Contains(mails[0].Body, event.GetVerificationCode())
}
