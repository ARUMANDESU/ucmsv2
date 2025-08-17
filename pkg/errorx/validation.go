package errorx

import (
	"errors"
	"strings"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

var ErrInvalidPasswordFormat = validation.NewError(
	"validation_is_password",
	"must be at least 8 characters long, contain at least one uppercase letter, one lowercase letter, one digit, and one special character",
)

// ValidatePassword validates a password string against the defined rules.
// It checks for minimum length, presence of uppercase, lowercase, digit, and special character.
func ValidatePasswordManual(value any) error {
	password, ok := value.(string)
	if !ok {
		return errors.New("value is not a string")
	}

	if len(password) < 8 {
		return ErrInvalidPasswordFormat
	}

	var hasLower, hasUpper, hasDigit, hasSpecial bool
	allowedSpecial := "@$!%*?&"

	for _, char := range password {
		switch {
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case strings.ContainsRune(allowedSpecial, char):
			hasSpecial = true
		default:
			// Invalid character found
			return ErrInvalidPasswordFormat
		}
	}

	if !hasLower || !hasUpper || !hasDigit || !hasSpecial {
		return ErrInvalidPasswordFormat
	}

	return nil
}

func ValidateGroupID(value any) error {
	groupID, ok := value.(uuid.UUID)
	if !ok {
		return errors.New("value is not a uuid.UUID")
	}

	if groupID == uuid.Nil {
		return validation.ErrRequired
	}

	return nil
}

func AssertValidationErrors(t *testing.T, err error, expected error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %v, got nil", expected)
	}

	var verrs validation.Errors
	if !errors.As(err, &verrs) {
		t.Fatalf("expected error to be of type *validation.Errors, got %T: %v", err, err)
	}

	var expectedVerrs validation.Errors
	if !errors.As(expected, &expectedVerrs) {
		t.Fatalf("expected expected error to be of type *validation.Errors, got %T: %v", expected, expected)
	}

	if verrs == nil || expectedVerrs == nil {
		t.Fatalf("expected non-nil validation errors, got %v and %v", verrs, expectedVerrs)
	}

	if len(verrs) != len(expectedVerrs) {
		t.Fatalf("expected number of validation errors to match, got %v and %v", verrs, expectedVerrs)
	}

	for field, expectedErr := range expectedVerrs {
		if actualErr, found := verrs[field]; !found {
			t.Errorf("field %s: expected error %v, got %v", field, expectedErr, actualErr)
		} else {
			AssertValidationError(t, actualErr, expectedErr)
		}
	}
}

func AssertValidationError(t *testing.T, err error, expected error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %v, got nil", expected)
	}

	var verrs validation.Error
	if !errors.As(err, &verrs) {
		t.Fatalf("expected error to be of type validation.Error, got %T: %v", err, err)
	}
	var expectedVerrs validation.Error
	if !errors.As(expected, &expectedVerrs) {
		t.Fatalf("expected expected error to be of type validation.Error, got %T: %v", expected, expected)
	}
	if verrs == nil || expectedVerrs == nil {
		t.Fatalf("expected non-nil validation error, got %v and %v", verrs, expectedVerrs)
	}

	if verrs.Code() != expectedVerrs.Code() || verrs.Message() != expectedVerrs.Message() {
		t.Errorf("expected validation error to match, got %v and %v", verrs, expectedVerrs)
	}
}
