package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/BurntSushi/toml"
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
		response := map[string]any{
			"code":    appErr.Code.String(),
			"message": appErr.Localize(localizer),
			"success": false,
		}

		err = WriteJSON(w, appErr.HTTPStatusCode(), response, nil)
		if err != nil {
			slog.ErrorContext(r.Context(), "Failed to write error response", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		return
	}

	slog.ErrorContext(r.Context(), "Unhandled error", "error", err)
	internalErr := errorx.ErrInternal.WithCause(err)
	response := map[string]any{
		"code":    internalErr.Code.String(),
		"message": internalErr.Localize(localizer),
		"success": false,
	}
	err = WriteJSON(w, internalErr.HTTPStatusCode(), response, nil)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to write internal error response", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func BadRequest(w http.ResponseWriter, r *http.Request, message string) {
	slog.ErrorContext(r.Context(), "Bad request", "message", message)
	response := map[string]any{
		"code":    errorx.CodeInvalid,
		"message": message,
		"success": false,
	}
	err := WriteJSON(w, http.StatusBadRequest, response, nil)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to write bad request response", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
