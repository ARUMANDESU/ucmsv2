package errorx

import (
	"fmt"
	"maps"
	"net/http"
	"strings"
	"testing"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/stretchr/testify/assert"
)

type I18nImmutableError struct {
	cause              error
	MessageKey         string
	MessageArgs        map[string]any
	MessagePluralCount any
	HTTPCode           int
	Code               Code
}

func (e I18nImmutableError) Error() string {
	if e.cause == nil {
		return fmt.Sprintf("[%s] %s", e.Code, e.MessageKey)
	}

	return fmt.Sprintf("[%s] %s: %s", e.Code, e.MessageKey, e.cause)
}

func (e I18nImmutableError) Unwrap() error {
	return e.cause
}

func (e *I18nImmutableError) Is(target error) bool {
	if target == nil {
		return e == nil
	}
	if targetErr, ok := target.(*I18nImmutableError); ok {
		fmt.Printf("Comparing events: %v and %v; result: %v\n", e, targetErr, e != nil && e.Code == targetErr.Code)
		return e != nil && e.Code == targetErr.Code
	}
	return false
}

func (e I18nImmutableError) Localize(localizer *i18n.Localizer) string {
	for key, value := range e.MessageArgs {
		if !strings.HasPrefix(key, "Locale_") {
			continue
		}
		if str, ok := value.(string); ok {
			e.MessageArgs[key] = localizer.MustLocalize(&i18n.LocalizeConfig{
				MessageID: str,
			})
		}
	}
	return localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID:    e.MessageKey,
		TemplateData: e.MessageArgs,
		PluralCount:  e.MessagePluralCount,
	})
}

func (e I18nImmutableError) HTTPStatusCode() int {
	if e.HTTPCode != 0 {
		return e.HTTPCode
	}

	return HTTPStatusCode(e.Code)
}

func (e I18nImmutableError) WithHTTPCode(code int) *I18nImmutableError {
	e.HTTPCode = code
	return &e
}

func (e I18nImmutableError) WithArgs(args map[string]any) *I18nImmutableError {
	if e.MessageArgs == nil {
		e.MessageArgs = make(map[string]any)
	}

	maps.Copy(e.MessageArgs, args)

	return &e
}

func (e I18nImmutableError) WithCause(cause error) *I18nImmutableError {
	e.cause = cause
	return &e
}

func (e I18nImmutableError) WithKey(key string) *I18nImmutableError {
	e.MessageKey = key
	return &e
}

func (e I18nImmutableError) WithCode(code Code) *I18nImmutableError {
	e.Code = code
	return &e
}

func NewI18nImmutableError(messageKey string) *I18nImmutableError {
	return &I18nImmutableError{
		MessageKey:  messageKey,
		MessageArgs: make(map[string]any),
		HTTPCode:    500,
		Code:        CodeInternal,
	}
}

var (
	ErrX = NewI18nImmutableError("x").WithHTTPCode(http.StatusTeapot)
	ErrY = NewI18nImmutableError("y").WithHTTPCode(http.StatusTeapot)
)

func TestImutableError(t *testing.T) {
	err := ErrX
	erry := ErrY

	diffErr := ErrX.WithKey("x_diff").WithArgs(map[string]any{"a": 1})
	diffErrY := ErrY.WithKey("y_diff").WithArgs(map[string]any{"b": 1})
	notErrY := ErrY.WithKey("y_diff_diff").WithArgs(map[string]any{"b": 2}).WithCode(CodeNotFound)

	assert.ErrorIs(t, diffErr, ErrX, "errors should be comparable with errors.Is")
	assert.ErrorIs(t, err, ErrX, "errors should be comparable with errors.Is")

	assert.ErrorIs(t, diffErrY, ErrY, "errors should be comparable with errors.Is")
	assert.ErrorIs(t, erry, ErrY, "errors should be comparable with errors.Is")
	assert.NotErrorIs(t, notErrY, ErrY, "errors with different codes should not be comparable with errors.Is")
	assert.NotErrorIs(t, notErrY, erry, "errors with different codes should not be comparable with errors.Is")

	assert.ErrorIs(t, err, erry)
	assert.ErrorIs(t, diffErr, diffErrY)
}
