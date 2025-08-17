package user

import (
	"strings"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/stretchr/testify/assert"

	"github.com/ARUMANDESU/ucms/pkg/errorx"
)

func TestUser_SetFirstName(t *testing.T) {
	tests := []struct {
		name        string
		user        *User
		firstName   string
		expectedErr error
	}{
		{
			name:        "valid first name",
			user:        &User{firstName: "OldFirstName"},
			firstName:   "NewFirstName",
			expectedErr: nil,
		},
		{
			name:        "empty first name",
			user:        &User{firstName: "OldFirstName"},
			firstName:   "",
			expectedErr: validation.ErrRequired,
		},
		{
			name:        "first name too short",
			user:        &User{firstName: "OldFirstName"},
			firstName:   "A",
			expectedErr: validation.ErrLengthOutOfRange,
		},
		{
			name:        "first name too long",
			user:        &User{firstName: "OldFirstName"},
			firstName:   "ThisIsAVeryLongFirstNameThatExceedsTheMaximumLengthThisIsAVeryLongFirstNameThatExceedsTheMaximumLengt",
			expectedErr: validation.ErrLengthOutOfRange,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.SetFirstName(tt.firstName)
			if (err != nil) != (tt.expectedErr != nil) {
				t.Errorf("User.SetFirstName() error = %v, expectedErr %v", err, tt.expectedErr)
			}
			if err != nil {
				errorx.AssertValidationError(t, err, tt.expectedErr)
			} else {
				assert.Equal(t, tt.firstName, tt.user.firstName, "First name should be set correctly")
			}
		})
	}
}

func TestUser_SetLastName(t *testing.T) {
	tests := []struct {
		name        string
		user        *User
		lastName    string
		expectedErr error
	}{
		{
			name:     "valid last name",
			user:     &User{lastName: "OldLastName"},
			lastName: "NewLastName",
		},
		{
			name:        "empty last name",
			user:        &User{lastName: "OldLastName"},
			lastName:    "",
			expectedErr: validation.ErrRequired,
		},
		{
			name:        "last name too short",
			user:        &User{lastName: "OldLastName"},
			lastName:    "A",
			expectedErr: validation.ErrLengthOutOfRange,
		},
		{
			name:        "last name too long",
			user:        &User{lastName: "OldLastName"},
			lastName:    "ThisIsAVeryLongLastNameThatExceedsTheMaximumLengthThisIsAVeryLongLastNameThatExceedsTheMaximumLengthT",
			expectedErr: validation.ErrLengthOutOfRange,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.SetLastName(tt.lastName)
			if (err != nil) != (tt.expectedErr != nil) {
				t.Errorf("User.SetLastName() error = %v, expectedErr %v", err, tt.expectedErr)
			}
			if err != nil {
				errorx.AssertValidationError(t, err, tt.expectedErr)
			} else {
				assert.Equal(t, tt.lastName, tt.user.lastName, "Last name should be set correctly")
			}
		})
	}
}

func TestUser_SetAvatarURL(t *testing.T) {
	tests := []struct {
		name        string
		user        *User
		avatarURL   string
		expectedErr error
	}{
		{
			name:      "valid avatar URL",
			user:      &User{avatarURL: "http://old-avatar.com/avatar.png"},
			avatarURL: "http://new-avatar.com/avatar.png",
		},
		{
			name:      "valid empty avatar URL",
			user:      &User{avatarURL: "http://old-avatar.com/avatar.png"},
			avatarURL: "",
		},
		{
			name:        "long avatar URL",
			user:        &User{avatarURL: "http://old-avatar.com/avatar.png"},
			avatarURL:   strings.Repeat("a", MaxAvatarURLLen+1),
			expectedErr: validation.ErrLengthOutOfRange,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.SetAvatarURL(tt.avatarURL)
			if (err != nil) != (tt.expectedErr != nil) {
				t.Errorf("User.SetAvatarURL() error = %v, expectedErr %v", err, tt.expectedErr)
			}
			if err != nil {
				errorx.AssertValidationError(t, err, tt.expectedErr)
			} else {
				assert.Equal(t, tt.avatarURL, tt.user.avatarURL, "Avatar URL should be set correctly")
			}
		})
	}
}

func TestUser_ComparePassword(t *testing.T) {
	tests := []struct {
		name     string
		user     *User
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			user:     &User{passHash: []byte("$2a$12$HoLMDChGzw26WRGqAdzeL.ZzauFTKP5tg/5d5VSBLsQvUuEBFsvgG")},
			password: "password",
			wantErr:  false,
		},
		{
			name:     "invalid password",
			user:     &User{passHash: []byte("$2a$12$HoLMDChGzw26WRGqAdzeL.ZzauFTKP5tg/5d5VSBLsQvUuEBFsvgG")},
			password: "wrongpassword",
			wantErr:  true,
		},
		{
			name:     "empty password",
			user:     &User{passHash: []byte("$2a$12$HoLMDChGzw26WRGqAdzeL.ZzauFTKP5tg/5d5VSBLsQvUuEBFsvgG")},
			password: "",
			wantErr:  true, // Expect an error when password is empty
		},
		{
			name:     "empty user",
			user:     &User{passHash: []byte{}}, // Empty passHash
			password: "password",
			wantErr:  true, // Expect an error when passHash is empty
		},
		{
			name:     "empty password and empty user",
			user:     &User{passHash: []byte{}}, // Empty passHash
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
