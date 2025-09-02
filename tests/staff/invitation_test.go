package staff

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ARUMANDESU/ucms/internal/domain/user"
	staffhttp "github.com/ARUMANDESU/ucms/internal/ports/http/staff"
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

	s.HTTP.CreateStaffInvitation(t,
		staffhttp.CreateInvitationRequest{
			Recipients: []string{fixtures.ValidStaff2Email, fixtures.ValidStaff3Email},
			ValidFrom:  nil,
			ValidUntil: nil,
		},
		httpframework.WithStaff(t, staffUser.User().ID()),
	).AssertStatus(http.StatusCreated)
}

func (s *StaffInvitationSuite) seedStaff(t *testing.T, email string) *user.Staff {
	t.Helper()
	staffUser := s.Builder.User.Staff(email)

	s.DB.SeedStaff(t, staffUser)

	return staffUser
}
