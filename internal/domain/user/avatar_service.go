package user

import (
	"fmt"
	"time"

	"github.com/ARUMANDESU/validation"

	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/i18nx"
)

const (
	MinAvatarSize = 100             // 100 bytes
	MaxAvatarSize = 5 * 1024 * 1024 // 5 MB
)

var (
	ErrInvalidFileType = validation.NewError(i18nx.ValidationInvalidFileType, i18nx.MsgValidationInvalidFileTypeOther)
	ErrAvatarTooLarge  = validation.NewError(i18nx.ValidationFileSizeTooLarge, i18nx.MsgValidationFileSizeTooLargeOther).
				SetParams(map[string]any{i18nx.ArgThreshold: MaxAvatarSize / (1024 * 1024), i18nx.ArgUnit: "MB"})
	ErrAvatarTooSmall = validation.NewError(i18nx.ValidationFileSizeTooSmall, i18nx.MsgValidationFileSizeTooSmallOther).
				SetParams(map[string]any{i18nx.ArgThreshold: MinAvatarSize, i18nx.ArgUnit: "bytes"})
)

type AvatarService struct {
	s3BaseURL string
}

func NewAvatarService(s3BaseURL string) *AvatarService {
	return &AvatarService{
		s3BaseURL: s3BaseURL,
	}
}

func (s *AvatarService) ValidateAvatarFile(contentType string, size int64) error {
	const op = "user.AvatarService.ValidateAvatarFile"
	allowedContentTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}

	if !allowedContentTypes[contentType] {
		err := ErrInvalidFileType.SetParams(map[string]any{i18nx.ArgList: "image/jpeg, image/png, image/gif, image/webp"})
		return errorx.Wrap(err, op)
	}

	if size > MaxAvatarSize {
		return errorx.Wrap(ErrAvatarTooLarge, op)
	}
	if size < MinAvatarSize {
		return errorx.Wrap(ErrAvatarTooSmall, op)
	}

	return nil
}

func (s *AvatarService) BuildAvatarURL(s3Key string) string {
	return fmt.Sprintf("%s/%s", s.s3BaseURL, s3Key)
}

// GenerateS3Key generates a unique S3 key for the user's avatar based on their user ID and current timestamp.
func (s *AvatarService) GenerateS3Key(userID ID) string {
	return fmt.Sprintf("avatars/%s/%d", userID.String(), timestampMillis())
}

func timestampMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
