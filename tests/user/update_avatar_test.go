package user

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/valueobject/avatars"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/builders"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/fixtures"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/framework"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/framework/event"
	httpframework "gitlab.com/ucmsv2/ucms-backend/tests/integration/framework/http"
)

type UpdateAvatarSuite struct {
	framework.IntegrationTestSuite
}

func TestUserSuite(t *testing.T) {
	suite.Run(t, new(UpdateAvatarSuite))
}

func (s *UpdateAvatarSuite) TestUpdateUserAvatar_HappyPath() {
	t := s.T()
	u := builders.NewUserBuilder().Build()
	s.DB.SeedUser(t, u)

	tests := []struct {
		name string
		file []byte
	}{
		{
			name: "5MB file size", // max avatar image limit
			file: fixtures.ValidJPEGAvatar,
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

			e := event.RequireEventuallyEvent[*user.UserAvatarUpdated](t, s.Event, 5*time.Second)
			assert.Equal(t, u.ID(), e.UserID, "event user ID should match")
			assert.NotEmpty(t, e.NewAvatar, "event avatar S3 key should not be empty")
			assert.Empty(t, e.OldAvatar, "event old avatar S3 key should be empty")
			assert.Equal(t, dbUser.Avatar().S3Key, e.NewAvatar.S3Key, "event avatar S3 key should match DB")
			assert.Equal(t, avatars.SourceS3, e.NewAvatar.Source, "event avatar source should be S3")
		})
	}
}

func (s *UpdateAvatarSuite) TestUpdateUserAvatar_ValidFormats() {
	t := s.T()

	validFixtures := fixtures.GetValidAvatars()
	require.NotEmpty(t, validFixtures, "should have valid avatar fixtures")

	for _, fixture := range validFixtures {
		t.Run(fixture.Description, func(t *testing.T) {
			u := builders.NewUserBuilder().Build()
			s.DB.SeedUser(t, u)

			resp := s.HTTP.UpdateUserAvatarWithFile(
				t,
				"avatar.jpg",
				fixture.ContentType,
				fixture.Data,
				httpframework.WithStudent(t, u.ID()),
			)
			resp.AssertStatus(http.StatusOK)

			dbUser := s.DB.RequireUserExists(t, u.Email()).
				AssertUpdatedAtWithin(time.Now(), time.Minute).
				AssertAvatarNotEmpty().
				User()

			require.Equal(t, avatars.SourceS3, dbUser.Avatar().Source, "avatar source should be S3")

			s.S3.RequireFile(t, dbUser.Avatar().S3Key)

			e := event.RequireEventuallyEvent[*user.UserAvatarUpdated](t, s.Event, 5*time.Second)
			assert.Equal(t, u.ID(), e.UserID, "event user ID should match")
			assert.NotEmpty(t, e.NewAvatar, "event avatar S3 key should not be empty")
			assert.Empty(t, e.OldAvatar, "event old avatar S3 key should be empty")
			assert.Equal(t, dbUser.Avatar().S3Key, e.NewAvatar.S3Key, "event avatar S3 key should match DB")
			assert.Equal(t, avatars.SourceS3, e.NewAvatar.Source, "event avatar source should be S3")
		})
	}
}

func (s *UpdateAvatarSuite) TestUpdateUserAvatar_InvalidFormats() {
	t := s.T()
	u := builders.NewUserBuilder().Build()
	s.DB.SeedUser(t, u)

	invalidFixtures := fixtures.GetInvalidAvatars()
	require.NotEmpty(t, invalidFixtures, "should have invalid avatar fixtures")

	for _, fixture := range invalidFixtures {
		t.Run(fixture.Description, func(t *testing.T) {
			resp := s.HTTP.UpdateUserAvatarWithFile(
				t,
				"avatar.jpg",
				fixture.ContentType,
				fixture.Data,
				httpframework.WithStudent(t, u.ID()),
			)
			resp.AssertStatus(http.StatusBadRequest)
		})
	}
}

func (s *UpdateAvatarSuite) TestUpdateUserAvatar_SizeValidation() {
	t := s.T()

	tests := []struct {
		name           string
		fileData       []byte
		expectedStatus int
		description    string
	}{
		{
			name:           "minimum_size_boundary",
			fileData:       fixtures.CreateRandomJPEGWithSize(fixtures.MinAvatarSize),
			expectedStatus: http.StatusOK,
			description:    "exactly minimum size should pass",
		},
		{
			name:           "below_minimum_size",
			fileData:       fixtures.CreateRandomJPEGWithSize(fixtures.MinAvatarSize - 1),
			expectedStatus: http.StatusBadRequest,
			description:    "below minimum size should fail",
		},
		{
			name:           "maximum_size_boundary",
			fileData:       fixtures.CreateRandomJPEGWithSize(fixtures.MaxAvatarSize),
			expectedStatus: http.StatusOK,
			description:    "exactly maximum size should pass",
		},
		{
			name:           "above_maximum_size",
			fileData:       fixtures.CreateOversizedJPEG(),
			expectedStatus: http.StatusBadRequest,
			description:    "above maximum size should fail",
		},
		{
			name:           "medium_size",
			fileData:       fixtures.CreateRandomJPEGWithSize(1024 * 1024),
			expectedStatus: http.StatusOK,
			description:    "medium size (1MB) should pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := builders.NewUserBuilder().Build()
			s.DB.SeedUser(t, u)

			resp := s.HTTP.UpdateUserAvatarWithFile(
				t,
				"avatar.jpg",
				"image/jpeg",
				tt.fileData,
				httpframework.WithStudent(t, u.ID()),
			)
			resp.AssertStatus(tt.expectedStatus)

			if tt.expectedStatus == http.StatusOK {
				dbUser := s.DB.RequireUserExists(t, u.Email()).
					AssertUpdatedAtWithin(time.Now(), time.Minute).
					AssertAvatarNotEmpty().
					User()
				require.Equal(t, avatars.SourceS3, dbUser.Avatar().Source, "avatar source should be S3")

				s.S3.RequireFile(t, dbUser.Avatar().S3Key)

				e := event.RequireEventuallyEvent[*user.UserAvatarUpdated](t, s.Event, 5*time.Second)
				assert.Equal(t, u.ID(), e.UserID, "event user ID should match")
				assert.NotEmpty(t, e.NewAvatar, "event avatar S3 key should not be empty")
				assert.Empty(t, e.OldAvatar, "event old avatar S3 key should be empty")
				assert.Equal(t, dbUser.Avatar().S3Key, e.NewAvatar.S3Key, "event avatar S3 key should match DB")
				assert.Equal(t, avatars.SourceS3, e.NewAvatar.Source, "event avatar source should be S3")
			}
		})
	}
}

func (s *UpdateAvatarSuite) TestUpdateUserAvatar_Authentication() {
	t := s.T()
	u := builders.NewUserBuilder().Build()
	s.DB.SeedUser(t, u)

	validAvatar := fixtures.GetAvatarByKey("valid_jpeg")
	require.NotNil(t, validAvatar)

	tests := []struct {
		name           string
		auth           httpframework.RequestBuilderOptions
		expectedStatus int
	}{
		{
			name:           "unauthenticated",
			auth:           httpframework.WithAnon(),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "authenticated_student",
			auth:           httpframework.WithStudent(t, u.ID()),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "authenticated_staff",
			auth:           httpframework.WithStaff(t, u.ID()),
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := s.HTTP.UpdateUserAvatarWithFile(
				t,
				"avatar.jpg",
				validAvatar.ContentType,
				validAvatar.Data,
				tt.auth,
			)
			resp.AssertStatus(tt.expectedStatus)
		})
	}
}

func (s *UpdateAvatarSuite) TestUpdateUserAvatar_ContentTypeValidation() {
	t := s.T()
	u := builders.NewUserBuilder().Build()
	s.DB.SeedUser(t, u)

	tests := []struct {
		name           string
		contentType    string
		fileData       []byte
		expectedStatus int
	}{
		{
			name:           "valid_jpeg_content_type",
			contentType:    "image/jpeg",
			fileData:       fixtures.ValidJPEGAvatar,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid_png_content_type",
			contentType:    "image/png",
			fileData:       fixtures.ValidPNGAvatar,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid_gif_content_type",
			contentType:    "image/gif",
			fileData:       fixtures.ValidGIFAvatar,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid_webp_content_type",
			contentType:    "image/webp",
			fileData:       fixtures.ValidWebPAvatar,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid_pdf_content_type",
			contentType:    "application/pdf",
			fileData:       fixtures.ValidJPEGAvatar,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid_text_content_type",
			contentType:    "text/plain",
			fileData:       fixtures.ValidJPEGAvatar,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing_content_type",
			contentType:    "",
			fileData:       fixtures.ValidJPEGAvatar,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := s.HTTP.UpdateUserAvatarWithFile(
				t,
				"avatar.jpg",
				tt.contentType,
				tt.fileData,
				httpframework.WithStudent(t, u.ID()),
			)
			resp.AssertStatus(tt.expectedStatus)
		})
	}
}

func (s *UpdateAvatarSuite) TestUpdateUserAvatar_MissingFile() {
	t := s.T()
	u := builders.NewUserBuilder().Build()
	s.DB.SeedUser(t, u)

	resp := s.HTTP.UpdateUserAvatar(
		t,
		nil,
		httpframework.WithStudent(t, u.ID()),
	)
	resp.AssertStatus(http.StatusBadRequest)
}

func (s *UpdateAvatarSuite) TestUpdateUserAvatar_ExistingAvatarReplacement() {
	t := s.T()
	originalS3Key := fixtures.TestS3Keys[0]
	u := builders.NewUserBuilder().WithS3Avatar(originalS3Key).Build()
	s.DB.SeedUser(t, u)

	require.NotEmpty(t, originalS3Key, "student should have existing avatar")

	validAvatar := fixtures.GetAvatarByKey("valid_png")
	require.NotNil(t, validAvatar)

	resp := s.HTTP.UpdateUserAvatarWithFile(
		t,
		"new_avatar.png",
		validAvatar.ContentType,
		validAvatar.Data,
		httpframework.WithStudent(t, u.ID()),
	)
	resp.AssertStatus(http.StatusOK)

	dbUser := s.DB.RequireUserExists(t, u.Email()).
		AssertUpdatedAtWithin(time.Now(), time.Minute).
		AssertAvatarNotEmpty().
		User()
	require.Equal(t, avatars.SourceS3, dbUser.Avatar().Source, "avatar source should be S3")

	s.S3.RequireFile(t, dbUser.Avatar().S3Key)

	e := event.RequireEventuallyEvent[*user.UserAvatarUpdated](t, s.Event, 5*time.Second)
	assert.Equal(t, u.ID(), e.UserID, "event user ID should match")
	assert.NotEmpty(t, e.NewAvatar, "event avatar S3 key should not be empty")
	assert.NotEmpty(t, e.OldAvatar, "event old avatar S3 key should not be empty")
	assert.Equal(t, dbUser.Avatar().S3Key, e.NewAvatar.S3Key, "event avatar S3 key should match DB")
	assert.Equal(t, avatars.SourceS3, e.NewAvatar.Source, "event avatar source should be S3")

	assert.Equal(t, originalS3Key, e.OldAvatar.S3Key, "event old avatar S3 key should match original")
	assert.Equal(t, avatars.SourceS3, e.OldAvatar.Source, "event old avatar source should be S3")
	assert.NotEqual(t, originalS3Key, e.NewAvatar.S3Key, "new avatar S3 key should differ from original")

	s.S3.RequireEventuallyNoFile(t, originalS3Key)
}

func (s *UpdateAvatarSuite) TestDeleteUserAvatar_HappyPath() {
	t := s.T()

	u := builders.NewUserBuilder().Build()
	s.DB.SeedUser(t, u)

	validAvatar := fixtures.GetAvatarByKey("valid_png")
	require.NotNil(t, validAvatar)

	s.HTTP.UpdateUserAvatarWithFile(
		t,
		"new_avatar.png",
		validAvatar.ContentType,
		validAvatar.Data,
		httpframework.WithUserJWT(t, u.ID()),
	).
		RequireStatus(http.StatusOK)

	s.HTTP.DeleteUserAvatar(t, httpframework.WithUserJWT(t, u.ID())).
		RequireStatus(http.StatusOK)
	s.DB.RequireUserExists(t, u.Email()).
		AssertEmptyAvatar()

	e := event.RequireEventuallyEvent[*user.UserAvatarUpdated](t, s.Event, 5*time.Second)
	assert.Equal(t, u.ID(), e.UserID, "event user ID should match")
	assert.Empty(t, e.NewAvatar, "event avatar S3 key should be empty")
	assert.NotEmpty(t, e.OldAvatar, "event old avatar S3 key should not be empty")

	s.S3.RequireEventuallyNoFile(t, e.OldAvatar.S3Key)
}

func (s *UpdateAvatarSuite) TestDeleteUserAvatar_Authentication() {
	t := s.T()

	u := builders.NewUserBuilder().Build()
	s.DB.SeedUser(t, u)

	validAvatar := fixtures.GetAvatarByKey("valid_png")
	require.NotNil(t, validAvatar)

	tests := []struct {
		name           string
		auth           httpframework.RequestBuilderOptions
		expectedStatus int
	}{
		{
			name:           "unauthenticated",
			auth:           httpframework.WithAnon(),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "authenticated_student",
			auth:           httpframework.WithStudent(t, u.ID()),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "authenticated_staff",
			auth:           httpframework.WithStaff(t, u.ID()),
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.HTTP.UpdateUserAvatarWithFile(
				t,
				"new_avatar.png",
				validAvatar.ContentType,
				validAvatar.Data,
				httpframework.WithUserJWT(t, u.ID()),
			).
				RequireStatus(http.StatusOK)
			s.HTTP.DeleteUserAvatar(t, tt.auth).RequireStatus(tt.expectedStatus)
		})
	}
}

func (s *UpdateAvatarSuite) TestDeleteUserAvatar_NotFound() {
	t := s.T()

	u := builders.NewUserBuilder().WithEmptyAvatar().Build()
	s.DB.SeedUser(t, u)

	s.HTTP.DeleteUserAvatar(t, httpframework.WithUserJWT(t, u.ID())).
		RequireStatus(http.StatusNotFound)
}
