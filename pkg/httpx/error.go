package httpx

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ARUMANDESU/validation"
	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/text/language"

	ucmsv2 "gitlab.com/ucmsv2/ucms-backend"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/otelx"
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

	locales := []string{
		"locales/en.toml",
		"locales/kk.toml",
		"locales/ru.toml",
		"locales/validation.en.toml",
		"locales/validation.kk.toml",
		"locales/validation.ru.toml",
		"locales/fields.en.toml",
		"locales/fields.kk.toml",
		"locales/fields.ru.toml",
	}

	for _, locale := range locales {
		if _, err := bundle.LoadMessageFileFS(ucmsv2.Locales, locale); err != nil {
			panic(fmt.Sprintf("Failed to load locale file %s: %v", locale, err))
		}
	}

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

func (h *ErrorHandler) HandleError(w http.ResponseWriter, r *http.Request, span trace.Span, err error, message string) {
	otelx.RecordSpanError(span, err, message)

	lang := r.Header.Get("Accept-Language")
	localizer := h.Localizer(lang)

	var appErrs errorx.I18nErrors
	var appErr *errorx.I18nError
	var valErrs validation.Errors
	var valErr validation.Error

	switch {

	case errors.As(err, &appErrs):
		writeError(w, r, httpErrorResponse{
			Status:  appErrs.HTTPStatusCode(),
			Code:    appErrs.Code(),
			Message: appErrs.Localize(localizer),
		})
	case errors.As(err, &appErr):
		writeError(w, r, httpErrorResponse{
			Status:  appErr.HTTPStatusCode(),
			Code:    appErr.Code,
			Message: appErr.Localize(localizer),
			Details: appErr.Details,
		})
	case errors.As(err, &valErrs):
		var msg strings.Builder
		for field, fieldErr := range valErrs {
			localizedField, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: field})
			if err == nil {
				field = localizedField
			}

			if valErr, ok := fieldErr.(validation.Error); ok {
				msg.WriteString(fmt.Sprintf("%s %s; ", field, localizer.MustLocalize(&i18n.LocalizeConfig{
					MessageID:    valErr.Code(),
					TemplateData: valErr.Params(),
				})))
			} else {
				msg.WriteString(fmt.Sprintf("%s: %s; ", field, fieldErr.Error()))
			}
		}
		writeError(w, r, httpErrorResponse{
			Status:  http.StatusBadRequest,
			Code:    errorx.CodeValidationFailed,
			Message: msg.String(),
		})
	case errors.As(err, &valErr):
		writeError(w, r, httpErrorResponse{
			Status: http.StatusBadRequest,
			Code:   errorx.CodeValidationFailed,
			Message: localizer.MustLocalize(&i18n.LocalizeConfig{
				MessageID:    valErr.Code(),
				TemplateData: valErr.Params(),
			}),
		})
	default:
		slog.ErrorContext(r.Context(), "Unhandled error", "error", err)
		internalErr := errorx.NewInternalError().WithCause(err, "handle_error")
		writeError(w, r, httpErrorResponse{
			Status:  internalErr.HTTPStatusCode(),
			Code:    internalErr.Code,
			Message: internalErr.Localize(localizer),
		})
		return
	}

	slog.ErrorContext(r.Context(), "HTTP error response", "error", err.Error())
}

type httpErrorResponse struct {
	Status  int         `json:"-"`
	Success bool        `json:"success"`
	Code    errorx.Code `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Details string      `json:"details,omitempty"`
}

func (h *httpErrorResponse) Envelope() map[string]any {
	return map[string]any{
		"success": h.Success,
		"code":    h.Code,
		"message": h.Message,
		"details": h.Details,
	}
}

func writeError(w http.ResponseWriter, r *http.Request, res httpErrorResponse) {
	err := WriteJSON(w, res.Status, res.Envelope(), nil)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to write error response", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
