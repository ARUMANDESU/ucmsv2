package user_test

import (
	"errors"
	"testing"

	"github.com/ARUMANDESU/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/event"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/valueobject/avatars"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/validationx"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/builders"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/fixtures"
)

func TestUser_ComparePassword(t *testing.T) {
	tests := []struct {
		name     string
		user     *user.User
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			user:     builders.NewUserBuilder().WithPassHash([]byte("$2a$12$HoLMDChGzw26WRGqAdzeL.ZzauFTKP5tg/5d5VSBLsQvUuEBFsvgG")).Build(),
			password: "password",
			wantErr:  false,
		},
		{
			name:     "invalid password",
			user:     builders.NewUserBuilder().WithPassHash([]byte("$2a$12$HoLMDChGzw26WRGqAdzeL.ZzauFTKP5tg/5d5VSBLsQvUuEBFsvgG")).Build(),
			password: "wrongpassword",
			wantErr:  true,
		},
		{
			name:     "empty password",
			user:     builders.NewUserBuilder().WithPassHash([]byte("$2a$12$HoLMDChGzw26WRGqAdzeL.ZzauFTKP5tg/5d5VSBLsQvUuEBFsvgG")).Build(),
			password: "",
			wantErr:  true, // Expect an error when password is empty
		},
		{
			name:     "empty user",
			user:     builders.NewUserBuilder().WithPassHash([]byte{}).Build(), // Empty passHash
			password: "password",
			wantErr:  true, // Expect an error when passHash is empty
		},
		{
			name:     "empty password and empty user",
			user:     builders.NewUserBuilder().WithPassHash([]byte{}).Build(), // Empty passHash
			password: "",
			wantErr:  true, // Expect an error when both user and password are empty
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.user.ComparePassword(tt.password); (err != nil) != tt.wantErr {
				t.Errorf("User.ComparePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUser_SetAvatarFromS3(t *testing.T) {
	tests := []struct {
		name    string
		user    *user.User
		s3Key   string
		wantErr error
	}{
		{
			name:  "valid s3 key",
			user:  builders.NewUserBuilder().Build(),
			s3Key: fixtures.TestS3Keys[0],
		},
		{
			name:  "valid s3 key with user with existing avatar",
			user:  builders.UserWithValidAvatar().Build(),
			s3Key: fixtures.TestS3Keys[1],
		},
		{
			name:    "empty s3 key",
			user:    builders.NewUserBuilder().Build(),
			s3Key:   fixtures.EmptyAvatarS3Key,
			wantErr: validation.ErrRequired,
		},
		{
			name:    "too long s3 key",
			user:    builders.NewUserBuilder().Build(),
			s3Key:   fixtures.LongAvatarS3Key,
			wantErr: validation.ErrLengthOutOfRange,
		},
		{
			name:    "nil user",
			user:    nil,
			s3Key:   fixtures.TestS3Keys[0],
			wantErr: errors.New("user is nil"),
		},
		{
			name:    "s3 key with only spaces",
			user:    builders.NewUserBuilder().Build(),
			s3Key:   "     ",
			wantErr: validation.ErrRequired,
		},
		{
			name:    "s3 key with special characters",
			user:    builders.NewUserBuilder().Build(),
			s3Key:   "avatar@#$.jpg",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldAvatar := tt.user.Avatar()
			err := tt.user.SetAvatarFromS3(tt.s3Key)
			if tt.wantErr != nil {
				require.Error(t, err)
				if errors.As(err, &validation.ErrorObject{}) {
					validationx.AssertValidationError(t, err, tt.wantErr)
				} else {
					assert.ErrorContains(t, err, tt.wantErr.Error())
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, avatars.SourceS3, tt.user.Avatar().Source)
			assert.Equal(t, tt.s3Key, tt.user.Avatar().S3Key)
			assert.Equal(t, "", tt.user.Avatar().External)

			events := tt.user.GetUncommittedEvents()
			require.Len(t, events, 1)
			e := event.AssertSingleEvent[*user.UserAvatarUpdated](t, events)
			assert.Equal(t, tt.user.ID(), e.UserID)
			assert.Equal(t, tt.user.Avatar(), e.NewAvatar)
			assert.Equal(t, oldAvatar, e.OldAvatar)
		})
	}
}

func TestUser_DeleteAvatar(t *testing.T) {
	tests := []struct {
		name    string
		user    *user.User
		wantErr error
	}{
		{
			name: "user with S3 avatar",
			user: builders.UserWithValidAvatar().Build(),
		},
		{
			name: "user with external avatar",
			user: builders.UserWithExternalAvatar().Build(),
		},
		{
			name:    "user with no avatar",
			user:    builders.NewUserBuilder().Build(),
			wantErr: errorx.NewNotFound(),
		},
		{
			name:    "nil user",
			user:    nil,
			wantErr: errors.New("user is nil"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldAvatar := tt.user.Avatar()
			err := tt.user.DeleteAvatar()
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr.Error())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, avatars.SourceUnknown, tt.user.Avatar().Source)
			assert.Equal(t, "", tt.user.Avatar().S3Key)
			assert.Equal(t, "", tt.user.Avatar().External)

			events := tt.user.GetUncommittedEvents()
			require.Len(t, events, 1)
			e := event.AssertSingleEvent[*user.UserAvatarUpdated](t, events)
			assert.Equal(t, tt.user.ID(), e.UserID)
			assert.Equal(t, tt.user.Avatar(), e.NewAvatar)
			assert.Equal(t, oldAvatar, e.OldAvatar)
		})
	}
}
