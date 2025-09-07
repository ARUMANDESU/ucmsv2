package user

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/valueobject/avatars"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/builders"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/fixtures"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/framework"
	httpframework "gitlab.com/ucmsv2/ucms-backend/tests/integration/framework/http"
)

type UserSuite struct {
	framework.IntegrationTestSuite
}

func TestUserSuite(t *testing.T) {
	suite.Run(t, new(UserSuite))
}

func (s *UserSuite) TestUpdateUserAvatar_HappyPath() {
	t := s.T()
	u := builders.NewUserBuilder().Build()
	s.DB.SeedUser(t, u)

	tests := []struct {
		name string
		file io.Reader
	}{
		{
			name: "5MB file size", // max avatar image limit
			file: bytes.NewReader(fixtures.ValidJPEGAvatar),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.HTTP.UpdateUserAvatar(t, tt.file, httpframework.WithStudent(t, u.ID())).
				RequireStatus(http.StatusOK)

			dbUser := s.DB.RequireUserExists(t, u.Email()).
				AssertUpdatedAtWithin(time.Now(), time.Minute).
				AssertAvatarNotEmpty().
				User()
			require.Equal(t, avatars.SourceS3, dbUser.Avatar().Source, "avatar source should be S3")

			s.S3.RequireFile(t, dbUser.Avatar().S3Key)
		})
	}
}
