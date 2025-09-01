package staff

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ARUMANDESU/ucms/tests/integration/framework"
)

type StaffInvitationSuite struct {
	framework.IntegrationTestSuite
}

func TestStaffInvitationSuite(t *testing.T) {
	suite.Run(t, new(StaffInvitationSuite))
}

func (s *StaffInvitationSuite) TestCreate_HappyPath() {}
