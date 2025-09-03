package staff

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	mailevent "github.com/ARUMANDESU/ucms/internal/application/mail/event"
	"github.com/ARUMANDESU/ucms/internal/domain/group"
	"github.com/ARUMANDESU/ucms/internal/domain/staffinvitation"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	staffhttp "github.com/ARUMANDESU/ucms/internal/ports/http/staff"
	"github.com/ARUMANDESU/ucms/tests/integration/builders"
	"github.com/ARUMANDESU/ucms/tests/integration/fixtures"
	"github.com/ARUMANDESU/ucms/tests/integration/framework"
	httpframework "github.com/ARUMANDESU/ucms/tests/integration/framework/http"
)

type StaffInvitationSuite struct {
	framework.IntegrationTestSuite
}

func TestStaffInvitationSuite(t *testing.T) {
	suite.Run(t, new(StaffInvitationSuite))
}

func (s *StaffInvitationSuite) TestCreate_HappyPath() {
	t := s.T()

	staffUser := s.seedStaff(t, fixtures.TestStaff.Email)

	t.Run("two recipients, no validity period", func(t *testing.T) {
		s.HTTP.CreateStaffInvitation(t,
			staffhttp.CreateInvitationRequest{
				Recipients: []string{fixtures.ValidStaff2Email, fixtures.ValidStaff3Email},
				ValidFrom:  nil,
				ValidUntil: nil,
			},
			httpframework.WithStaff(t, staffUser.User().ID()),
		).AssertStatus(http.StatusCreated)

		s.MockMailSender.EventuallyRequireMailSent(t, fixtures.ValidStaff3Email, mailevent.StaffInvitationSubject)
		mail := s.MockMailSender.EventuallyRequireMailSent(t, fixtures.ValidStaff2Email, mailevent.StaffInvitationSubject)
		assert.Contains(t, mail.Body, "Please use the following link to accept the invitation:")

		code := parseCodeFromMailBody(t, mail.Body)

		s.DB.RequireStaffInvitationExistsByCode(t, code).
			AssertRecipientsEmail([]string{fixtures.ValidStaff2Email, fixtures.ValidStaff3Email}).
			AssertValidFrom(nil).
			AssertValidUntil(nil).
			AssertCreatorID(staffUser.User().ID())
	})

	t.Run("duplicate recipients", func(t *testing.T) {
		email := randomEmail()
		s.HTTP.CreateStaffInvitation(t,
			staffhttp.CreateInvitationRequest{
				Recipients: []string{email, email},
				ValidFrom:  nil,
				ValidUntil: nil,
			},
			httpframework.WithStaff(t, staffUser.User().ID()),
		).AssertStatus(http.StatusCreated)

		mail := s.MockMailSender.EventuallyRequireMailSent(t, email, mailevent.StaffInvitationSubject)
		assert.Contains(t, mail.Body, "Please use the following link to accept the invitation:")
		code := parseCodeFromMailBody(t, mail.Body)
		s.DB.RequireStaffInvitationExistsByCode(t, code).
			AssertRecipientsEmail([]string{email}).
			AssertValidFrom(nil).
			AssertValidUntil(nil).
			AssertCreatorID(staffUser.User().ID())
	})

	t.Run("empty recipients", func(t *testing.T) {
		s.HTTP.CreateStaffInvitation(t,
			staffhttp.CreateInvitationRequest{
				Recipients: []string{},
				ValidFrom:  nil,
				ValidUntil: nil,
			},
			httpframework.WithStaff(t, staffUser.User().ID()),
		).AssertStatus(http.StatusCreated)

		// TODO: somehow verify invitation created with no recipients, no email sent

		s.DB.RequireLatestStaffInvitationByCreatorID(t, staffUser.User().ID()).
			AssertRecipientsEmail([]string{}).
			AssertValidFrom(nil).
			AssertValidUntil(nil).
			AssertCreatorID(staffUser.User().ID())
	})

	t.Run("one recipient, no validity period", func(t *testing.T) {
		s.HTTP.CreateStaffInvitation(t,
			staffhttp.CreateInvitationRequest{
				Recipients: []string{fixtures.ValidStaff4Email},
				ValidFrom:  nil,
				ValidUntil: nil,
			},
			httpframework.WithStaff(t, staffUser.User().ID()),
		).AssertStatus(http.StatusCreated)

		mail := s.MockMailSender.EventuallyRequireMailSent(t, fixtures.ValidStaff4Email, mailevent.StaffInvitationSubject)
		assert.Contains(t, mail.Body, "Please use the following link to accept the invitation:")

		code := parseCodeFromMailBody(t, mail.Body)

		s.DB.RequireStaffInvitationExistsByCode(t, code).
			AssertRecipientsEmail([]string{fixtures.ValidStaff4Email}).
			AssertValidFrom(nil).
			AssertValidUntil(nil).
			AssertCreatorID(staffUser.User().ID())
	})

	t.Run("one recipient, with validity period", func(t *testing.T) {
		validFrom := time.Now().AddDate(0, 0, 1).Truncate(time.Second).UTC() // from tomorrow
		validUntil := validFrom.AddDate(0, 0, 7).Truncate(time.Second).UTC() // for one week
		email := randomEmail()
		s.HTTP.CreateStaffInvitation(t,
			staffhttp.CreateInvitationRequest{
				Recipients: []string{email},
				ValidFrom:  &validFrom,
				ValidUntil: &validUntil,
			},
			httpframework.WithStaff(t, staffUser.User().ID()),
		).AssertStatus(http.StatusCreated)

		mail := s.MockMailSender.EventuallyRequireMailSent(t, email, mailevent.StaffInvitationSubject)
		assert.Contains(t, mail.Body, "Please use the following link to accept the invitation:")

		code := parseCodeFromMailBody(t, mail.Body)
		s.DB.RequireStaffInvitationExistsByCode(t, code).
			AssertRecipientsEmail([]string{email}).
			AssertValidFrom(&validFrom).
			AssertValidUntil(&validUntil).
			AssertCreatorID(staffUser.User().ID())
	})

	t.Run("one recipient, with validFrom only", func(t *testing.T) {
		validFrom := time.Now().AddDate(0, 0, 1).Truncate(time.Second).UTC() // from tomorrow
		email := randomEmail()
		s.HTTP.CreateStaffInvitation(t,
			staffhttp.CreateInvitationRequest{
				Recipients: []string{email},
				ValidFrom:  &validFrom,
				ValidUntil: nil,
			},
			httpframework.WithStaff(t, staffUser.User().ID()),
		).AssertStatus(http.StatusCreated)

		mail := s.MockMailSender.EventuallyRequireMailSent(t, email, mailevent.StaffInvitationSubject)
		assert.Contains(t, mail.Body, "Please use the following link to accept the invitation:")
		code := parseCodeFromMailBody(t, mail.Body)
		s.DB.RequireStaffInvitationExistsByCode(t, code).
			AssertRecipientsEmail([]string{email}).
			AssertValidFrom(&validFrom).
			AssertValidUntil(nil).
			AssertCreatorID(staffUser.User().ID())
	})

	t.Run("one recipient, with validUntil only", func(t *testing.T) {
		validUntil := time.Now().AddDate(0, 0, 7).Truncate(time.Second).UTC() // for one week
		email := randomEmail()
		s.HTTP.CreateStaffInvitation(t,
			staffhttp.CreateInvitationRequest{
				Recipients: []string{email},
				ValidFrom:  nil,
				ValidUntil: &validUntil,
			},
			httpframework.WithStaff(t, staffUser.User().ID()),
		).AssertStatus(http.StatusCreated)

		mail := s.MockMailSender.EventuallyRequireMailSent(t, email, mailevent.StaffInvitationSubject)
		assert.Contains(t, mail.Body, "Please use the following link to accept the invitation:")
		code := parseCodeFromMailBody(t, mail.Body)
		s.DB.RequireStaffInvitationExistsByCode(t, code).
			AssertRecipientsEmail([]string{email}).
			AssertValidFrom(nil).
			AssertValidUntil(&validUntil).
			AssertCreatorID(staffUser.User().ID())
	})
}

func (s *StaffInvitationSuite) TestCreate_FailPath() {
	t := s.T()

	staffUser := s.seedStaff(t, fixtures.TestStaff.Email)
	groupID := s.seedGroup(t)
	studentUser := s.seedStudent(t, fixtures.TestStudent.Email, groupID)
	authOpts := []httpframework.RequestBuilderOptions{
		httpframework.WithStaff(t, staffUser.User().ID()),
	}

	tests := []struct {
		name    string
		request staffhttp.CreateInvitationRequest
		opts    []httpframework.RequestBuilderOptions
		assert  func(t *testing.T, resp *httpframework.Response)
	}{
		{
			name: "unauthenticated",
			request: staffhttp.CreateInvitationRequest{
				Recipients: []string{fixtures.ValidStaff2Email},
				ValidFrom:  nil,
				ValidUntil: nil,
			},
			opts: []httpframework.RequestBuilderOptions{httpframework.WithAnon()},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusUnauthorized)
			},
		},
		{
			name: "forbidden for non-staff user",
			request: staffhttp.CreateInvitationRequest{
				Recipients: []string{fixtures.ValidStaff2Email},
				ValidFrom:  nil,
				ValidUntil: nil,
			},
			opts: []httpframework.RequestBuilderOptions{
				httpframework.WithStudent(t, studentUser.User().ID()),
			},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusForbidden)
			},
		},
		{
			name: "invalid email in recipients",
			request: staffhttp.CreateInvitationRequest{
				Recipients: []string{"invalid-email"},
				ValidFrom:  nil,
				ValidUntil: nil,
			},
			opts: authOpts,
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage("Recipients' Email: 0: must be a valid email address")
			},
		},
		{
			name: "validUntil before validFrom",
			request: staffhttp.CreateInvitationRequest{
				Recipients: []string{fixtures.ValidStaff2Email},
				ValidFrom:  ptrToTime(time.Now().AddDate(0, 0, 7).Truncate(time.Second).UTC()), // 7 days from now
				ValidUntil: ptrToTime(time.Now().AddDate(0, 0, 1).Truncate(time.Second).UTC()), // 1 day from now
			},
			opts: authOpts,
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage("valid_until time must be after")
			},
		},
		{
			name: "validFrom in the past",
			request: staffhttp.CreateInvitationRequest{
				Recipients: []string{fixtures.ValidStaff2Email},
				ValidFrom:  ptrToTime(time.Now().Add(-1 * time.Hour).Truncate(time.Second).UTC()), // 1 hour ago
				ValidUntil: nil,
			},
			opts: authOpts,
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage("valid_from time cannot be in the past")
			},
		},
		{
			name: "validUntil in the past",
			request: staffhttp.CreateInvitationRequest{
				Recipients: []string{fixtures.ValidStaff2Email},
				ValidFrom:  nil,
				ValidUntil: ptrToTime(time.Now().Add(-1 * time.Hour).Truncate(time.Second).UTC()), // 1 hour ago
			},
			opts: authOpts,
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage("valid_until time cannot be in the past")
			},
		},
		{
			name: "too many recipients (over the limit)",
			request: staffhttp.CreateInvitationRequest{
				Recipients: func() []string {
					emails := make([]string, staffinvitation.MaxEmails+1) // One over the limit
					for i := range staffinvitation.MaxEmails + 1 {
						emails[i] = randomEmail()
					}
					return emails
				}(),
				ValidFrom:  nil,
				ValidUntil: nil,
			},
			opts: authOpts,
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage(fmt.Sprintf("the count must be no more than %d", staffinvitation.MaxEmails))
			},
		},
		{
			name: "empty email in recipients list",
			request: staffhttp.CreateInvitationRequest{
				Recipients: []string{fixtures.ValidStaff2Email, ""},
				ValidFrom:  nil,
				ValidUntil: nil,
			},
			opts: authOpts,
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage("Recipients' Email: 1: cannot be blank")
			},
		},
		{
			name: "mixed valid and invalid emails",
			request: staffhttp.CreateInvitationRequest{
				Recipients: []string{fixtures.ValidStaff2Email, "invalid-email", fixtures.ValidStaff3Email},
				ValidFrom:  nil,
				ValidUntil: nil,
			},
			opts: authOpts,
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage("Recipients' Email: 1: must be a valid email address")
			},
		},
		{
			name: "email with special characters",
			request: staffhttp.CreateInvitationRequest{
				Recipients: []string{"test+special@example.com"},
				ValidFrom:  nil,
				ValidUntil: nil,
			},
			opts: authOpts,
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusCreated)
			},
		},
		{
			name: "very long email address",
			request: staffhttp.CreateInvitationRequest{
				Recipients: []string{strings.Repeat("a", 250) + "@example.com"}, // Over typical email length
				ValidFrom:  nil,
				ValidUntil: nil,
			},
			opts: authOpts,
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage("Recipients' Email: 0: must be a valid email address")
			},
		},
		{
			name: "email without domain",
			request: staffhttp.CreateInvitationRequest{
				Recipients: []string{"justusername"},
				ValidFrom:  nil,
				ValidUntil: nil,
			},
			opts: authOpts,
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage("Recipients' Email: 0: must be a valid email address")
			},
		},
		{
			name: "email with only @ symbol",
			request: staffhttp.CreateInvitationRequest{
				Recipients: []string{"@"},
				ValidFrom:  nil,
				ValidUntil: nil,
			},
			opts: authOpts,
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage("Recipients' Email: 0: must be a valid email address")
			},
		},
		{
			name: "recipients list with whitespace-only email",
			request: staffhttp.CreateInvitationRequest{
				Recipients: []string{"   "},
				ValidFrom:  nil,
				ValidUntil: nil,
			},
			opts: authOpts,
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage("Recipients' Email: 0: cannot be blank")
			},
		},
		{
			name: "validFrom exactly equal to validUntil",
			request: staffhttp.CreateInvitationRequest{
				Recipients: []string{fixtures.ValidStaff2Email},
				ValidFrom:  ptrToTime(time.Now().AddDate(0, 0, 1).Truncate(time.Second).UTC()),
				ValidUntil: ptrToTime(time.Now().AddDate(0, 0, 1).Truncate(time.Second).UTC()), // Same time as validFrom
			},
			opts: authOpts,
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage("valid_until time must be after")
			},
		},
		{
			name: "maximum valid recipients (exactly at the limit)",
			request: staffhttp.CreateInvitationRequest{
				Recipients: func() []string {
					emails := make([]string, staffinvitation.MaxEmails)
					for i := range staffinvitation.MaxEmails {
						emails[i] = randomEmail()
					}
					return emails
				}(),
				ValidFrom:  nil,
				ValidUntil: nil,
			},
			opts: authOpts,
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusCreated)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := s.HTTP.CreateStaffInvitation(t, tc.request, tc.opts...)
			tc.assert(t, resp)
		})
	}
}

func (s *StaffInvitationSuite) TestUpdateRecipients_HappyPath() {
	t := s.T()

	staffUser := s.seedStaff(t, fixtures.TestStaff.Email)

	t.Run("update recipients of existing invitation", func(t *testing.T) {
		invitation := builders.NewStaffInvitationBuilder().
			WithRecipientsEmail([]string{fixtures.ValidStaff2Email}).
			WithCreatorID(staffUser.User().ID()).
			Build()
		s.DB.SeedStaffInvitation(t, invitation)

		newEmail := randomEmail()
		s.HTTP.UpdateStaffInvitationRecipients(t, invitation.ID().String(),
			staffhttp.UpdateInvitationRecipientsRequest{
				Recipients: []string{fixtures.ValidStaff3Email, newEmail},
			},
			httpframework.WithStaff(t, staffUser.User().ID()),
		).AssertStatus(http.StatusOK)

		s.MockMailSender.EventuallyRequireMailSent(t, fixtures.ValidStaff3Email, mailevent.StaffInvitationSubject)
		s.MockMailSender.EventuallyRequireMailSent(t, newEmail, mailevent.StaffInvitationSubject)

		s.DB.RequireStaffInvitationExists(t, invitation.ID()).
			AssertRecipientsEmail([]string{fixtures.ValidStaff3Email, newEmail}).
			AssertCreatorID(staffUser.User().ID())
	})

	t.Run("update to empty recipients list", func(t *testing.T) {
		invitation := builders.NewStaffInvitationBuilder().
			WithRecipientsEmail([]string{fixtures.ValidStaff2Email}).
			WithCreatorID(staffUser.User().ID()).
			Build()
		s.DB.SeedStaffInvitation(t, invitation)

		s.HTTP.UpdateStaffInvitationRecipients(t, invitation.ID().String(),
			staffhttp.UpdateInvitationRecipientsRequest{
				Recipients: []string{},
			},
			httpframework.WithStaff(t, staffUser.User().ID()),
		).AssertStatus(http.StatusOK)

		s.DB.RequireStaffInvitationExists(t, invitation.ID()).
			AssertRecipientsEmail([]string{}).
			AssertCreatorID(staffUser.User().ID())
	})

	t.Run("update recipients with duplicates", func(t *testing.T) {
		invitation := builders.NewStaffInvitationBuilder().
			WithRecipientsEmail([]string{fixtures.ValidStaff2Email}).
			WithCreatorID(staffUser.User().ID()).
			Build()
		s.DB.SeedStaffInvitation(t, invitation)

		email := randomEmail()
		s.HTTP.UpdateStaffInvitationRecipients(t, invitation.ID().String(),
			staffhttp.UpdateInvitationRecipientsRequest{
				Recipients: []string{email, email, fixtures.ValidStaff3Email},
			},
			httpframework.WithStaff(t, staffUser.User().ID()),
		).AssertStatus(http.StatusOK)

		s.DB.RequireStaffInvitationExists(t, invitation.ID()).
			AssertRecipientsEmail([]string{email, fixtures.ValidStaff3Email}).
			AssertCreatorID(staffUser.User().ID())
	})
}

func (s *StaffInvitationSuite) TestUpdateRecipients_FailPath() {
	t := s.T()

	staffUser := s.seedStaff(t, fixtures.TestStaff.Email)
	groupID := s.seedGroup(t)
	studentUser := s.seedStudent(t, fixtures.TestStudent.Email, groupID)
	invitation := builders.NewStaffInvitationBuilder().
		WithRecipientsEmail([]string{fixtures.ValidStaff2Email}).
		WithCreatorID(staffUser.User().ID()).
		Build()
	s.DB.SeedStaffInvitation(t, invitation)

	tests := []struct {
		name         string
		invitationID string
		request      staffhttp.UpdateInvitationRecipientsRequest
		opts         []httpframework.RequestBuilderOptions
		assert       func(t *testing.T, resp *httpframework.Response)
	}{
		{
			name:         "unauthenticated",
			invitationID: invitation.ID().String(),
			request: staffhttp.UpdateInvitationRecipientsRequest{
				Recipients: []string{fixtures.ValidStaff3Email},
			},
			opts: []httpframework.RequestBuilderOptions{httpframework.WithAnon()},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusUnauthorized)
			},
		},
		{
			name:         "forbidden for non-staff user",
			invitationID: invitation.ID().String(),
			request: staffhttp.UpdateInvitationRecipientsRequest{
				Recipients: []string{fixtures.ValidStaff3Email},
			},
			opts: []httpframework.RequestBuilderOptions{
				httpframework.WithStudent(t, studentUser.User().ID()),
			},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusForbidden)
			},
		},
		{
			name:         "invitation not found",
			invitationID: staffinvitation.NewID().String(),
			request: staffhttp.UpdateInvitationRecipientsRequest{
				Recipients: []string{fixtures.ValidStaff3Email},
			},
			opts: []httpframework.RequestBuilderOptions{
				httpframework.WithStaff(t, staffUser.User().ID()),
			},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusNotFound)
			},
		},
		{
			name:         "invalid email in recipients",
			invitationID: invitation.ID().String(),
			request: staffhttp.UpdateInvitationRecipientsRequest{
				Recipients: []string{"invalid-email"},
			},
			opts: []httpframework.RequestBuilderOptions{
				httpframework.WithStaff(t, staffUser.User().ID()),
			},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage("Recipients' Email: 0: must be a valid email address")
			},
		},
		{
			name:         "too many recipients",
			invitationID: invitation.ID().String(),
			request: staffhttp.UpdateInvitationRecipientsRequest{
				Recipients: func() []string {
					emails := make([]string, staffinvitation.MaxEmails+1) // One over the limit
					for i := range staffinvitation.MaxEmails + 1 {
						emails[i] = randomEmail()
					}
					return emails
				}(),
			},
			opts: []httpframework.RequestBuilderOptions{
				httpframework.WithStaff(t, staffUser.User().ID()),
			},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage(fmt.Sprintf("the count must be no more than %d", staffinvitation.MaxEmails))
			},
		},
		{
			name:         "empty email in recipients list",
			invitationID: invitation.ID().String(),
			request: staffhttp.UpdateInvitationRecipientsRequest{
				Recipients: []string{fixtures.ValidStaff3Email, ""},
			},
			opts: []httpframework.RequestBuilderOptions{
				httpframework.WithStaff(t, staffUser.User().ID()),
			},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage("Recipients' Email: 1: cannot be blank")
			},
		},
		{
			name:         "very long email address",
			invitationID: invitation.ID().String(),
			request: staffhttp.UpdateInvitationRecipientsRequest{
				Recipients: []string{strings.Repeat("a", 250) + "@example.com"},
			},
			opts: []httpframework.RequestBuilderOptions{
				httpframework.WithStaff(t, staffUser.User().ID()),
			},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage("Recipients' Email: 0: must be a valid email address")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := s.HTTP.UpdateStaffInvitationRecipients(t, tc.invitationID, tc.request, tc.opts...)
			tc.assert(t, resp)
		})
	}
}

func (s *StaffInvitationSuite) TestUpdateValidity_HappyPath() {
	t := s.T()

	staffUser := s.seedStaff(t, fixtures.TestStaff.Email)

	t.Run("update validity period of existing invitation", func(t *testing.T) {
		invitation := builders.NewStaffInvitationBuilder().
			WithRecipientsEmail([]string{fixtures.ValidStaff2Email}).
			WithCreatorID(staffUser.User().ID()).
			Build()
		s.DB.SeedStaffInvitation(t, invitation)

		validFrom := time.Now().AddDate(0, 0, 1).Truncate(time.Second).UTC()
		validUntil := validFrom.AddDate(0, 0, 7).Truncate(time.Second).UTC()

		s.HTTP.UpdateStaffInvitationValidity(t, invitation.ID().String(),
			staffhttp.UpdateInvitationValidityRequest{
				ValidFrom:  &validFrom,
				ValidUntil: &validUntil,
			},
			httpframework.WithStaff(t, staffUser.User().ID()),
		).AssertStatus(http.StatusOK)

		s.DB.RequireStaffInvitationExists(t, invitation.ID()).
			AssertValidFrom(&validFrom).
			AssertValidUntil(&validUntil).
			AssertCreatorID(staffUser.User().ID())
	})

	t.Run("clear validity period", func(t *testing.T) {
		validFrom := time.Now().AddDate(0, 0, 1).Truncate(time.Second).UTC()
		validUntil := validFrom.AddDate(0, 0, 7).Truncate(time.Second).UTC()
		invitation := builders.NewStaffInvitationBuilder().
			WithRecipientsEmail([]string{fixtures.ValidStaff2Email}).
			WithValidFrom(&validFrom).
			WithValidUntil(&validUntil).
			WithCreatorID(staffUser.User().ID()).
			Build()
		s.DB.SeedStaffInvitation(t, invitation)

		s.HTTP.UpdateStaffInvitationValidity(t, invitation.ID().String(),
			staffhttp.UpdateInvitationValidityRequest{
				ValidFrom:  nil,
				ValidUntil: nil,
			},
			httpframework.WithStaff(t, staffUser.User().ID()),
		).AssertStatus(http.StatusOK)

		s.DB.RequireStaffInvitationExists(t, invitation.ID()).
			AssertValidFrom(nil).
			AssertValidUntil(nil).
			AssertCreatorID(staffUser.User().ID())
	})

	t.Run("update only validFrom", func(t *testing.T) {
		invitation := builders.NewStaffInvitationBuilder().
			WithRecipientsEmail([]string{fixtures.ValidStaff2Email}).
			WithCreatorID(staffUser.User().ID()).
			Build()
		s.DB.SeedStaffInvitation(t, invitation)

		validFrom := time.Now().AddDate(0, 0, 2).Truncate(time.Second).UTC()

		s.HTTP.UpdateStaffInvitationValidity(t, invitation.ID().String(),
			staffhttp.UpdateInvitationValidityRequest{
				ValidFrom:  &validFrom,
				ValidUntil: nil,
			},
			httpframework.WithStaff(t, staffUser.User().ID()),
		).AssertStatus(http.StatusOK)

		s.DB.RequireStaffInvitationExists(t, invitation.ID()).
			AssertValidFrom(&validFrom).
			AssertValidUntil(nil).
			AssertCreatorID(staffUser.User().ID())
	})
}

func (s *StaffInvitationSuite) TestUpdateValidity_FailPath() {
	t := s.T()

	staffUser := s.seedStaff(t, fixtures.TestStaff.Email)
	groupID := s.seedGroup(t)
	studentUser := s.seedStudent(t, fixtures.TestStudent.Email, groupID)

	invitation := builders.NewStaffInvitationBuilder().
		WithRecipientsEmail([]string{fixtures.ValidStaff2Email}).
		WithCreatorID(staffUser.User().ID()).
		Build()
	s.DB.SeedStaffInvitation(t, invitation)

	tests := []struct {
		name         string
		invitationID string
		request      staffhttp.UpdateInvitationValidityRequest
		opts         []httpframework.RequestBuilderOptions
		assert       func(t *testing.T, resp *httpframework.Response)
	}{
		{
			name:         "unauthenticated",
			invitationID: invitation.ID().String(),
			request: staffhttp.UpdateInvitationValidityRequest{
				ValidFrom:  ptrToTime(time.Now().AddDate(0, 0, 1).Truncate(time.Second).UTC()),
				ValidUntil: nil,
			},
			opts: []httpframework.RequestBuilderOptions{httpframework.WithAnon()},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusUnauthorized)
			},
		},
		{
			name:         "forbidden for non-staff user",
			invitationID: invitation.ID().String(),
			request: staffhttp.UpdateInvitationValidityRequest{
				ValidFrom:  ptrToTime(time.Now().AddDate(0, 0, 1).Truncate(time.Second).UTC()),
				ValidUntil: nil,
			},
			opts: []httpframework.RequestBuilderOptions{
				httpframework.WithStudent(t, studentUser.User().ID()),
			},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusForbidden)
			},
		},
		{
			name:         "invitation not found",
			invitationID: staffinvitation.NewID().String(),
			request: staffhttp.UpdateInvitationValidityRequest{
				ValidFrom:  ptrToTime(time.Now().AddDate(0, 0, 1).Truncate(time.Second).UTC()),
				ValidUntil: nil,
			},
			opts: []httpframework.RequestBuilderOptions{
				httpframework.WithStaff(t, staffUser.User().ID()),
			},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusNotFound)
			},
		},
		{
			name:         "validFrom in the past",
			invitationID: invitation.ID().String(),
			request: staffhttp.UpdateInvitationValidityRequest{
				ValidFrom:  ptrToTime(time.Now().Add(-1 * time.Hour).Truncate(time.Second).UTC()),
				ValidUntil: nil,
			},
			opts: []httpframework.RequestBuilderOptions{
				httpframework.WithStaff(t, staffUser.User().ID()),
			},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage("time cannot be in the past")
			},
		},
		{
			name:         "validUntil in the past",
			invitationID: invitation.ID().String(),
			request: staffhttp.UpdateInvitationValidityRequest{
				ValidFrom:  nil,
				ValidUntil: ptrToTime(time.Now().Add(-1 * time.Hour).Truncate(time.Second).UTC()),
			},
			opts: []httpframework.RequestBuilderOptions{
				httpframework.WithStaff(t, staffUser.User().ID()),
			},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage("time cannot be in the past")
			},
		},
		{
			name:         "validUntil before validFrom",
			invitationID: invitation.ID().String(),
			request: staffhttp.UpdateInvitationValidityRequest{
				ValidFrom:  ptrToTime(time.Now().AddDate(0, 0, 7).Truncate(time.Second).UTC()),
				ValidUntil: ptrToTime(time.Now().AddDate(0, 0, 1).Truncate(time.Second).UTC()),
			},
			opts: []httpframework.RequestBuilderOptions{
				httpframework.WithStaff(t, staffUser.User().ID()),
			},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage("time must be after")
			},
		},
		{
			name:         "validFrom exactly equal to validUntil",
			invitationID: invitation.ID().String(),
			request: staffhttp.UpdateInvitationValidityRequest{
				ValidFrom:  ptrToTime(time.Now().AddDate(0, 0, 1).Truncate(time.Second).UTC()),
				ValidUntil: ptrToTime(time.Now().AddDate(0, 0, 1).Truncate(time.Second).UTC()),
			},
			opts: []httpframework.RequestBuilderOptions{
				httpframework.WithStaff(t, staffUser.User().ID()),
			},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusBadRequest).
					AssertContainsMessage("time must be after")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := s.HTTP.UpdateStaffInvitationValidity(t, tc.invitationID, tc.request, tc.opts...)
			tc.assert(t, resp)
		})
	}
}

func (s *StaffInvitationSuite) TestDeleteInvitation_HappyPath() {
	t := s.T()

	staffUser := s.seedStaff(t, fixtures.TestStaff.Email)

	t.Run("delete existing invitation", func(t *testing.T) {
		invitation := builders.NewStaffInvitationBuilder().
			WithRecipientsEmail([]string{fixtures.ValidStaff2Email}).
			WithCreatorID(staffUser.User().ID()).Build()
		s.DB.SeedStaffInvitation(t, invitation)

		s.HTTP.DeleteStaffInvitation(t, invitation.ID().String(),
			httpframework.WithStaff(t, staffUser.User().ID()),
		).AssertStatus(http.StatusOK)

		s.DB.RequireStaffInvitationExists(t, invitation.ID()).AssertDeleted(true)
	})

	t.Run("delete invitation with validity period", func(t *testing.T) {
		validFrom := time.Now().AddDate(0, 0, 1).Truncate(time.Second).UTC()
		validUntil := validFrom.AddDate(0, 0, 7).Truncate(time.Second).UTC()

		invitation := builders.NewStaffInvitationBuilder().
			WithRecipientsEmail([]string{fixtures.ValidStaff3Email}).
			WithValidFrom(&validFrom).
			WithValidUntil(&validUntil).
			WithCreatorID(staffUser.User().ID()).Build()
		s.DB.SeedStaffInvitation(t, invitation)

		s.HTTP.DeleteStaffInvitation(t, invitation.ID().String(),
			httpframework.WithStaff(t, staffUser.User().ID()),
		).AssertStatus(http.StatusOK)

		s.DB.RequireStaffInvitationExists(t, invitation.ID()).AssertDeleted(true)
	})

	t.Run("delete already deleted invitation", func(t *testing.T) {
		invitation := builders.NewStaffInvitationBuilder().
			WithRecipientsEmail([]string{fixtures.ValidStaff4Email}).
			WithDeletedAt(ptrToTime(time.Now().Add(-1 * time.Hour).Truncate(time.Second).UTC())).
			WithCreatorID(staffUser.User().ID()).Build()
		s.DB.SeedStaffInvitation(t, invitation)

		s.HTTP.DeleteStaffInvitation(t, invitation.ID().String(),
			httpframework.WithStaff(t, staffUser.User().ID()),
		).AssertStatus(http.StatusOK)

		s.DB.RequireStaffInvitationExists(t, invitation.ID()).AssertDeleted(true)
	})
}

func (s *StaffInvitationSuite) TestDeleteInvitation_FailPath() {
	t := s.T()

	staffUser := s.seedStaff(t, fixtures.TestStaff.Email)
	groupID := s.seedGroup(t)
	studentUser := s.seedStudent(t, fixtures.TestStudent.Email, groupID)

	invitation := builders.NewStaffInvitationBuilder().
		WithRecipientsEmail([]string{fixtures.ValidStaff2Email}).
		WithCreatorID(staffUser.User().ID()).Build()
	s.DB.SeedStaffInvitation(t, invitation)

	tests := []struct {
		name         string
		invitationID string
		opts         []httpframework.RequestBuilderOptions
		assert       func(t *testing.T, resp *httpframework.Response)
	}{
		{
			name:         "unauthenticated",
			invitationID: invitation.ID().String(),
			opts:         []httpframework.RequestBuilderOptions{httpframework.WithAnon()},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusUnauthorized)
			},
		},
		{
			name:         "forbidden for non-staff user",
			invitationID: invitation.ID().String(),
			opts: []httpframework.RequestBuilderOptions{
				httpframework.WithStudent(t, studentUser.User().ID()),
			},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusForbidden)
			},
		},
		{
			name:         "invitation not found",
			invitationID: uuid.NewString(),
			opts: []httpframework.RequestBuilderOptions{
				httpframework.WithStaff(t, staffUser.User().ID()),
			},
			assert: func(t *testing.T, resp *httpframework.Response) {
				resp.AssertStatus(http.StatusNotFound)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := s.HTTP.DeleteStaffInvitation(t, tc.invitationID, tc.opts...)
			tc.assert(t, resp)
		})
	}
}

func parseCodeFromMailBody(t *testing.T, body string) string {
	t.Helper()
	// Example body: "Please use the following link to accept the invitation: <URL>/<CODE>?email=..."
	parts := strings.Split(body, "/")
	if len(parts) < 2 {
		t.Fatalf("Failed to parse code from mail body: %s", body)
	}
	codeAndQuery := parts[len(parts)-1]
	code := strings.Split(codeAndQuery, "?")[0]
	return code
}

func ptrToTime(t time.Time) *time.Time {
	return &t
}

func randomEmail() string {
	return strings.ToLower(uuid.NewString()[:8] + "@test.com")
}

func (s *StaffInvitationSuite) seedStaff(t *testing.T, email string) *user.Staff {
	t.Helper()
	staffUser := s.Builder.User.Staff(email)
	s.DB.SeedStaff(t, staffUser)
	return staffUser
}

func (s *StaffInvitationSuite) seedGroup(t *testing.T) group.ID {
	t.Helper()
	groupID := group.NewID()
	s.DB.SeedGroup(t, groupID, fixtures.SEGroup.Name, fixtures.SEGroup.Year, fixtures.SEGroup.Major)
	return groupID
}

func (s *StaffInvitationSuite) seedStudent(t *testing.T, email string, groupID group.ID) *user.Student {
	t.Helper()
	studentUser := builders.NewStudentBuilder().WithEmail(email).WithGroupID(groupID).Build()
	s.DB.SeedStudent(t, studentUser)
	return studentUser
}
