package staff

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/ARUMANDESU/ucms/internal/domain/staffinvitation"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
	staffhttp "github.com/ARUMANDESU/ucms/internal/ports/http/staff"
	"github.com/ARUMANDESU/ucms/tests/integration/builders"
	"github.com/ARUMANDESU/ucms/tests/integration/fixtures"
	"github.com/ARUMANDESU/ucms/tests/integration/framework"
	"github.com/ARUMANDESU/ucms/tests/integration/framework/event"
	httpframework "github.com/ARUMANDESU/ucms/tests/integration/framework/http"
)

type AcceptInvitationTest struct {
	framework.IntegrationTestSuite
}

func TestAcceptInvitation(t *testing.T) {
	suite.Run(t, new(AcceptInvitationTest))
}

func (s *AcceptInvitationTest) TestVerify_HappyPath() {
	t := s.T()

	staffUser := s.SeedStaff(t, fixtures.TestStaff.Email)

	tests := []struct {
		name       string
		invitation *staffinvitation.StaffInvitation
		email      string
	}{
		{
			name: "valid code and email",
			invitation: builders.NewStaffInvitationBuilder().
				WithCreatorID(staffUser.User().ID()).
				Build(),
			email: randomEmail(),
		},
		{
			name: "valid code and one of multiple emails",
			invitation: builders.NewStaffInvitationBuilder().
				WithCreatorID(staffUser.User().ID()).
				WithRecipientsEmail([]string{randomEmail(), randomEmail(), randomEmail()}).
				Build(),
			email: randomEmail(),
		},
		{
			name: "valid code and full recipient list",
			invitation: builders.NewStaffInvitationBuilder().
				WithCreatorID(staffUser.User().ID()).
				WithRecipientsEmail(randomEmails(staffinvitation.MaxEmails - 1)).
				Build(),
			email: randomEmail(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invitation := builders.NewStaffInvitationBuilder().
				WithCreatorID(staffUser.User().ID()).
				WithAppendRecipientsEmail(tt.email).
				Build()
			s.DB.SeedStaffInvitation(t, invitation)

			resp := s.HTTP.ValidateStaffInvitation(t, invitation.Code(), tt.email, httpframework.WithStaff(t, staffUser.User().ID())).
				RequireStatus(http.StatusFound).
				AssertHeaderContains("Location", fixtures.StaffInvitationAcceptPageURL)
			AssertLocation(t, resp, invitation, tt.email)
		})
	}
}

func (s *AcceptInvitationTest) TestVerify_FailPath() {
	t := s.T()

	staffUser := s.SeedStaff(t, fixtures.TestStaff.Email)
	validEmail := randomEmail()
	invalidEmail := randomEmail()
	invalidFormatEmail := "invalid-email-format"
	validCode := fixtures.StaffInvitationValidCode
	invalidCode := fixtures.StaffInvitationInvalidCode
	invitation := builders.NewStaffInvitationBuilder().
		WithCreatorID(staffUser.User().ID()).
		WithRecipientsEmail([]string{validEmail}).
		Build()
	s.DB.SeedStaffInvitation(t, invitation)

	tests := []struct {
		name           string
		code           string
		email          string
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "invalid code, but valid email",
			code:           invalidCode,
			email:          validEmail,
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Resource not found or has been deleted",
		},
		{
			name:           "valid code, but email not in recipient list",
			code:           validCode,
			email:          invalidEmail,
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Resource not found or has been deleted",
		},
		{
			name:           "valid code, but invalid email format",
			code:           validCode,
			email:          invalidFormatEmail,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "must be a valid email address",
		},
		{
			name:           "valid code, but empty email",
			code:           validCode,
			email:          "",
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "cannot be blank",
		},
		{
			name:           "empty code and email",
			code:           "",
			email:          "",
			expectedStatus: http.StatusNotFound, // because code is in the path and empty code does not match any route
		},
		{
			name:           "empty code, but valid email",
			code:           "",
			email:          validEmail,
			expectedStatus: http.StatusNotFound, // because code is in the path and empty code does not match any route
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := s.HTTP.ValidateStaffInvitation(t, tt.code, tt.email).RequireStatus(tt.expectedStatus)
			if tt.expectedMsg != "" {
				resp.AssertContainsMessage(tt.expectedMsg)
			}
			require.Empty(t, resp.Header().Get("Location"))
		})
	}
}

func (s *AcceptInvitationTest) TestAccept_HappyPath() {
	t := s.T()

	staffUser := s.SeedStaff(t, fixtures.TestStaff.Email)
	email := randomEmail()
	invitation := builders.NewStaffInvitationBuilder().
		WithCreatorID(staffUser.User().ID()).
		WithAppendRecipientsEmail(email).
		Build()
	s.DB.SeedStaffInvitation(t, invitation)

	token, err := staffhttp.SignInvitationJWTToken(
		invitation.Code(),
		email,
		fixtures.InvitationTokenAlg,
		fixtures.InvitationTokenKey,
		fixtures.InvitationTokenExp,
	)
	require.NoError(t, err)

	s.HTTP.AcceptStaffInvitation(t, staffhttp.AcceptInvitationRequest{
		Token:     token,
		Barcode:   fixtures.TestStaff2.Barcode.String(),
		Username:  fixtures.TestStaff2.Username,
		Password:  fixtures.TestStaff2.Password,
		FirstName: fixtures.TestStaff2.FirstName,
		LastName:  fixtures.TestStaff2.LastName,
	}).
		RequireStatus(http.StatusCreated)

	staffAssertion := s.DB.RequireStaffExistsByEmail(t, email).
		AssertIDNotEmpty(t).
		AssertBarcode(t, fixtures.TestStaff2.Barcode).
		AssertUsername(t, fixtures.TestStaff2.Username).
		AssertFirstName(t, fixtures.TestStaff2.FirstName).
		AssertLastName(t, fixtures.TestStaff2.LastName).
		AssertPassword(t, fixtures.TestStaff2.Password).
		AssertRole(t, role.Staff)

	e := event.RequireEvent(t, s.Event, &user.StaffInvitationAccepted{})
	user.NewStaffInvitationAcceptedAssertion(t, e).
		AssertStaffID(staffAssertion.Staff().User().ID()).
		AssertStaffBarcode(fixtures.TestStaff2.Barcode).
		AssertStaffUsername(fixtures.TestStaff2.Username).
		AssertFirstName(fixtures.TestStaff2.FirstName).
		AssertLastName(fixtures.TestStaff2.LastName).
		AssertInvitationID(uuid.UUID(invitation.ID())).
		AssertEmail(email)
}

func (s *AcceptInvitationTest) TestAccept_FailPath() {
	t := s.T()

	staffUser := s.SeedStaff(t, fixtures.TestStaff.Email)
	email := randomEmail()
	invalidEmail := randomEmail()
	invitation := builders.NewStaffInvitationBuilder().
		WithCreatorID(staffUser.User().ID()).
		WithAppendRecipientsEmail(email).
		Build()
	s.DB.SeedStaffInvitation(t, invitation)

	validToken, err := staffhttp.SignInvitationJWTToken(
		invitation.Code(),
		email,
		fixtures.InvitationTokenAlg,
		fixtures.InvitationTokenKey,
		fixtures.InvitationTokenExp,
	)
	require.NoError(t, err)

	invalidToken, err := staffhttp.SignInvitationJWTToken(
		invitation.Code(),
		invalidEmail,
		fixtures.InvitationTokenAlg,
		fixtures.InvitationTokenKey,
		fixtures.InvitationTokenExp,
	)
	require.NoError(t, err)

	tests := []struct {
		name           string
		req            staffhttp.AcceptInvitationRequest
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "invalid token (email not in recipient list)",
			req: staffhttp.AcceptInvitationRequest{
				Token:     invalidToken,
				Barcode:   fixtures.TestStaff2.Barcode.String(),
				Username:  fixtures.TestStaff2.Username,
				Password:  fixtures.TestStaff2.Password,
				FirstName: fixtures.TestStaff2.FirstName,
				LastName:  fixtures.TestStaff2.LastName,
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid invitation or does not exist",
		},
		{
			name: "empty token",
			req: staffhttp.AcceptInvitationRequest{
				Token:     "",
				Barcode:   fixtures.TestStaff2.Barcode.String(),
				Username:  fixtures.TestStaff2.Username,
				Password:  fixtures.TestStaff2.Password,
				FirstName: fixtures.TestStaff2.FirstName,
				LastName:  fixtures.TestStaff2.LastName,
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "cannot be blank",
		},
		{
			name: "empty barcode",
			req: staffhttp.AcceptInvitationRequest{
				Token:     validToken,
				Barcode:   "",
				Username:  fixtures.TestStaff2.Username,
				Password:  fixtures.TestStaff2.Password,
				FirstName: fixtures.TestStaff2.FirstName,
				LastName:  fixtures.TestStaff2.LastName,
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "cannot be blank",
		},
		{
			name: "empty username",
			req: staffhttp.AcceptInvitationRequest{
				Token:     validToken,
				Barcode:   fixtures.TestStaff2.Barcode.String(),
				Username:  "",
				Password:  fixtures.TestStaff2.Password,
				FirstName: fixtures.TestStaff2.FirstName,
				LastName:  fixtures.TestStaff2.LastName,
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "cannot be blank",
		},
		{
			name: "empty password",
			req: staffhttp.AcceptInvitationRequest{
				Token:     validToken,
				Barcode:   fixtures.TestStaff2.Barcode.String(),
				Username:  fixtures.TestStaff2.Username,
				Password:  "",
				FirstName: fixtures.TestStaff2.FirstName,
				LastName:  fixtures.TestStaff2.LastName,
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "cannot be blank",
		},
		{
			name: "empty first name",
			req: staffhttp.AcceptInvitationRequest{
				Token:     validToken,
				Barcode:   fixtures.TestStaff2.Barcode.String(),
				Username:  fixtures.TestStaff2.Username,
				Password:  fixtures.TestStaff2.Password,
				FirstName: "",
				LastName:  fixtures.TestStaff2.LastName,
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "cannot be blank",
		},
		{
			name: "empty last name",
			req: staffhttp.AcceptInvitationRequest{
				Token:     validToken,
				Barcode:   fixtures.TestStaff2.Barcode.String(),
				Username:  fixtures.TestStaff2.Username,
				Password:  fixtures.TestStaff2.Password,
				FirstName: fixtures.TestStaff2.FirstName,
				LastName:  "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "cannot be blank",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := s.HTTP.AcceptStaffInvitation(t, tt.req).
				RequireStatus(tt.expectedStatus)
			if tt.expectedMsg != "" {
				resp.AssertContainsMessage(tt.expectedMsg)
			}
			s.DB.RequireStaffNotExistsByEmail(t, email)
		})
	}
}

func AssertLocation(t *testing.T, resp *httpframework.Response, invitation *staffinvitation.StaffInvitation, email string) {
	t.Helper()

	location := resp.Header().Get("Location")
	require.NotEmpty(t, location)
	token := parseTokenFromLocation(t, location)

	jwtInvitationCode, jwtEmail, err := staffhttp.ParseInvitationJWTToken(token, fixtures.InvitationTokenAlg, fixtures.InvitationTokenKey)
	require.NoError(t, err)
	require.Equal(t, invitation.Code(), jwtInvitationCode)
	require.Equal(t, email, jwtEmail)
}

func parseTokenFromLocation(t *testing.T, location string) string {
	t.Helper()
	parsedURL, err := url.Parse(location)
	require.NoError(t, err, "failed to parse location URL: %s", location)
	token := parsedURL.Query().Get("token")
	require.NotEmpty(t, token, "token not found in location URL: %s", location)
	return token
}

func randomEmails(count int) []string {
	emails := make([]string, count)
	for i := range count {
		emails[i] = randomEmail()
	}
	return emails
}
