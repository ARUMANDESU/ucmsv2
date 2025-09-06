package user_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/pkg/validationx"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/fixtures"
)

func TestAvatarService_ValidateAvatarFile(t *testing.T) {
	s := newAvatarService()

	tests := []struct {
		name        string
		contentType string
		size        int64
		wantErr     error
	}{
		{
			name:        "valid jpeg",
			contentType: "image/jpeg",
			size:        1024 * 1024,
			wantErr:     nil,
		},
		{
			name:        "valid png",
			contentType: "image/png",
			size:        1024 * 1024,
			wantErr:     nil,
		},
		{
			name:        "valid gif",
			contentType: "image/gif",
			size:        1024 * 1024,
			wantErr:     nil,
		},
		{
			name:        "valid webp",
			contentType: "image/webp",
			size:        1024 * 1024,
			wantErr:     nil,
		},
		{
			name:        "invalid content type",
			contentType: "application/pdf",
			size:        1024 * 1024,
			wantErr:     user.ErrInvalidFileType,
		},
		{
			name:        "too large file",
			contentType: "image/jpeg",
			size:        user.MaxAvatarSize + 1,
			wantErr:     user.ErrAvatarTooLarge,
		},
		{
			name:        "zero size file",
			contentType: "image/png",
			size:        0,
			wantErr:     user.ErrAvatarTooSmall,
		},
		{
			name:        "too small file",
			contentType: "image/png",
			size:        user.MinAvatarSize - 1,
			wantErr:     user.ErrAvatarTooSmall,
		},
		{
			name:        "boundary size file - min",
			contentType: "image/png",
			size:        user.MinAvatarSize,
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.ValidateAvatarFile(tt.contentType, tt.size)
			if tt.wantErr != nil {
				validationx.AssertValidationError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAvatarService_BuildAvatarURL(t *testing.T) {
	s := newAvatarService()

	s3Key := "avatars/user123/avatar.png"
	expectedURL := fixtures.ValidS3BaseURL + "/" + s3Key

	url := s.BuildAvatarURL(s3Key)
	require.Equal(t, expectedURL, url)
}

func TestAvatarService_GenerateS3Key(t *testing.T) {
	s := newAvatarService()

	userID := user.NewID()
	s3Key := s.GenerateS3Key(userID)

	require.Contains(t, s3Key, "avatars/"+userID.String()+"/")
}

func newAvatarService() *user.AvatarService {
	return user.NewAvatarService(fixtures.ValidS3BaseURL)
}
