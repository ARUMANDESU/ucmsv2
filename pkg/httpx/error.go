package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/ARUMANDESU/ucms/pkg/apperr"
)

func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("error handling request", "error", err)
	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		switch appErr.HTTPStatusCode() {
		case http.StatusUnauthorized:
			Unauthorized(w, r, appErr.Message)
		case http.StatusBadRequest:
			BadRequest(w, r, appErr.Message)
		case http.StatusNotFound:
			NotFound(w, r, appErr.Message)
		case http.StatusForbidden:
			Forbidden(w, r, appErr.Message)
		case http.StatusConflict:
			Conflict(w, r, appErr.Message)
		case http.StatusInternalServerError:
			InternalServerError(w, r)
		default:
			InternalServerError(w, r)
		}

		return
	}

	InternalServerError(w, r)
}

func Error(w http.ResponseWriter, r *http.Request, status int, errStr string, message string) {
	slog.Error("error", "status", status, "error", errStr, "message", message)
	response := map[string]any{
		"code":    errStr,
		"message": message,
		"success": false,
	}

	err := WriteJSON(w, status, response, nil)
	if err != nil {
		// log.Error("failed to write error response", zap.Error(err))
	}
}

func Unauthorized(w http.ResponseWriter, r *http.Request, message string) {
	Error(w, r, http.StatusUnauthorized, "unauthorized", message)
}

func BadRequest(w http.ResponseWriter, r *http.Request, message string) {
	Error(w, r, http.StatusBadRequest, "bad-request", message)
}

func NotFound(w http.ResponseWriter, r *http.Request, message string) {
	Error(w, r, http.StatusNotFound, "not-found", message)
}

func Conflict(w http.ResponseWriter, r *http.Request, message string) {
	Error(w, r, http.StatusConflict, "conflict", message)
}

func Forbidden(w http.ResponseWriter, r *http.Request, message string) {
	Error(w, r, http.StatusForbidden, "forbidden", message)
}

func InternalServerError(w http.ResponseWriter, r *http.Request) {
	Error(w, r, http.StatusInternalServerError, "internal-server-error", "internal server error")
}
