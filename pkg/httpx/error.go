package httpx

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/BurntSushi/toml"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"

	ucmsv2 "github.com/ARUMANDESU/ucms"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
)

type ErrorHandler struct {
	bundle *i18n.Bundle
	enloc  *i18n.Localizer
	kkloc  *i18n.Localizer
	ruloc  *i18n.Localizer
}

func NewErrorHandler() *ErrorHandler {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	// Load translation files
	bundle.LoadMessageFileFS(ucmsv2.Locales, "locales/en.toml")
	bundle.LoadMessageFileFS(ucmsv2.Locales, "locales/kk.toml")
	bundle.LoadMessageFileFS(ucmsv2.Locales, "locales/ru.toml")

	// Load validation files
	bundle.LoadMessageFileFS(ucmsv2.Locales, "locales/validation.en.toml")
	bundle.LoadMessageFileFS(ucmsv2.Locales, "locales/validation.kk.toml")
	bundle.LoadMessageFileFS(ucmsv2.Locales, "locales/validation.ru.toml")

	return &ErrorHandler{
		bundle: bundle,
		enloc:  i18n.NewLocalizer(bundle, "en"),
		kkloc:  i18n.NewLocalizer(bundle, "kk"),
		ruloc:  i18n.NewLocalizer(bundle, "ru"),
	}
}

func (h *ErrorHandler) Localizer(lang string) *i18n.Localizer {
	switch lang {
	case "kk":
		return h.kkloc
	case "ru":
		return h.ruloc
	default:
		return h.enloc
	}
}

func (h *ErrorHandler) HandleError(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "HTTP error response", "error", err.Error())

	lang := r.Header.Get("Accept-Language")
	localizer := h.Localizer(lang)

	var appErr *errorx.I18nError
	if errors.As(err, &appErr) {
		writeError(w, r,
			appErr.Code,
			appErr.Localize(localizer),
			appErr.HTTPStatusCode(),
		)
		return
	}

	var valErrs validation.Errors
	if errors.As(err, &valErrs) {
		var msg strings.Builder
		for field, fieldErr := range valErrs {
			if valErr, ok := fieldErr.(validation.Error); ok {
				msg.WriteString(fmt.Sprintf("%s: %s; ", field, localizer.MustLocalize(&i18n.LocalizeConfig{
					MessageID:    valErr.Code(),
					TemplateData: valErr.Params(),
				})))
			} else {
				msg.WriteString(fmt.Sprintf("%s: %s; ", field, fieldErr.Error()))
			}
		}
		writeError(w, r,
			errorx.CodeValidationFailed,
			msg.String(),
			http.StatusBadRequest,
		)
		return
	}

	var valErr validation.Error
	if errors.As(err, &valErr) {
		writeError(w, r,
			errorx.CodeValidationFailed,
			localizer.MustLocalize(&i18n.LocalizeConfig{
				MessageID:    valErr.Code(),
				TemplateData: valErr.Params(),
			}),
			http.StatusBadRequest,
		)
		return
	}

	slog.ErrorContext(r.Context(), "Unhandled error", "error", err)
	internalErr := errorx.NewInternalError().WithCause(err)
	writeError(w, r,
		internalErr.Code,
		internalErr.Localize(localizer),
		internalErr.HTTPStatusCode(),
	)
}

func BadRequest(w http.ResponseWriter, r *http.Request, message string) {
	slog.ErrorContext(r.Context(), "Bad request", "message", message)
	writeError(w, r,
		errorx.CodeInvalid,
		message,
		http.StatusBadRequest,
	)
}

func writeError(w http.ResponseWriter, r *http.Request,
	code errorx.Code,
	message string,
	status int,
) {
	response := map[string]any{
		"code":    code,
		"message": message,
		"success": false,
	}

	err := WriteJSON(w, status, response, nil)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to write error response", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
