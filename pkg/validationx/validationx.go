package validationx

import (
	"errors"
	"reflect"
	"regexp"
	"testing"
	"time"
	"unicode"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var ErrInvalidPasswordFormat = validation.NewError(
	"validation_is_password",
	"must be at least 8 characters long, contain at least one uppercase letter, one lowercase letter, one digit, and one special character",
)

var ErrInvalidNameFormat = validation.NewError(
	"validation_is_name",
	"must be a valid name containing only letters, spaces, hyphens, apostrophes, and periods")

var (
	PasswordFormat = PasswordFormatRule{}
	// Required is a validation rule that checks if a value is not empty. Use it for uuid verification, otherwise use validation.Required.
	Required = RequiredRule{}
)

var (
	// Allow Unicode letters, spaces, hyphens, apostrophes, periods
	nameRegex  = regexp.MustCompile(`^[\p{L}\p{M}\s'\-\.]+$`)
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	// Allow alphanumeric characters
	barcodeRegex = regexp.MustCompile(`^[A-Z0-9]{6,20}$`)
)

var IsPersonName = validation.By(func(value interface{}) error {
	s, _ := value.(string)
	if s == "" {
		return nil // Let Required handle emptiness
	}

	if !nameRegex.MatchString(s) {
		return errors.New("must be a valid name")
	}
	return nil
})

type PasswordFormatRule struct{}

// Validate validates a password string against the defined rules.
// It checks for minimum length, presence of uppercase, lowercase, digit, and special character.
func (r PasswordFormatRule) Validate(value any) error {
	password, ok := value.(string)
	if !ok {
		return errors.New("value is not a string")
	}

	if len(password) < 8 {
		return ErrInvalidPasswordFormat
	}

	var hasLower, hasUpper, hasDigit, hasSpecial bool

	for _, char := range password {
		switch {
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
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

type RequiredRule struct{}

func (r RequiredRule) Validate(value any) error {
	value, isNil := validation.Indirect(value)
	if isNil || isEmpty(value) {
		return validation.ErrRequired
	}

	return nil
}

func isEmpty(value any) bool {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Array:
		return v.Equal(reflect.Zero(v.Type())) || v.Len() == 0
	case reflect.String:
		return v.Len() == 0 || v.String() == "" || v.String() == "00000000-0000-0000-0000-000000000000" // for uuid empty string
	case reflect.Map, reflect.Slice:
		return v.Equal(reflect.Zero(v.Type())) || v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Invalid:
		return true
	case reflect.Interface, reflect.Ptr:
		if v.IsNil() {
			return true
		}
		return isEmpty(v.Elem().Interface())
	case reflect.Struct:
		v, ok := value.(time.Time)
		if ok && v.IsZero() {
			return true
		}
	}

	return false
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

func IsEmailOrBarcode(emailbarcode string) (isEmail bool, isBarcode bool) {
	return emailRegex.MatchString(emailbarcode), barcodeRegex.MatchString(emailbarcode)
}
