package s3helper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/ucmsv2/ucms-backend/internal/adapters/services/s3"
)

type Helper struct {
	s3 *s3.Client
}

func NewHelper(s3Client *s3.Client) *Helper {
	if s3Client == nil {
		panic("s3 client is required")
	}

	return &Helper{
		s3: s3Client,
	}
}

func (h *Helper) RequireFile(t *testing.T, key string) {
	t.Helper()

	_, err := h.s3.GetObject(t.Context(), key)
	require.NoError(t, err, "failed to get file from S3")
}

func (h *Helper) RequireNoFile(t *testing.T, key string) {
	t.Helper()

	_, err := h.s3.GetObject(t.Context(), key)
	require.Error(t, err, "expected error when getting non-existing file from S3")
}

func (h *Helper) RequireEventuallyNoFile(t *testing.T, key string) {
	t.Helper()

	require.Eventually(t, func() bool {
		_, err := h.s3.GetObject(t.Context(), key)
		return err != nil
	}, 5*time.Second, 100*time.Millisecond, "file still exists in S3")
}
