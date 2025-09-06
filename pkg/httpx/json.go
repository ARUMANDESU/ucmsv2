package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
)

type Envelope map[string]any

const maxRequestBodySize = 10 << 20 // 10MB

func ReadJSON(w http.ResponseWriter, r *http.Request, v any) error {
	const op = "httpx.ReadJSON"
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(v)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		malformedErr := errorx.NewMalformedJSON().WithCause(err, op)
		switch {
		case errors.As(err, &syntaxError):
			_ = malformedErr.WithDetails(fmt.Sprintf("badly-formed JSON (at character %d)", syntaxError.Offset))
		case errors.Is(err, io.ErrUnexpectedEOF):
			_ = malformedErr.WithDetails("body contains badly-formed JSON")
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				_ = malformedErr.WithDetails(
					fmt.Sprintf("body contains incorrect JSON type for field %q (at character %d)",
						unmarshalTypeError.Field,
						unmarshalTypeError.Offset,
					),
				)
			} else {
				_ = malformedErr.WithDetails(fmt.Sprintf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset))
			}
		case errors.Is(err, io.EOF):
			_ = malformedErr.WithDetails("body must not be empty")
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			_ = malformedErr.WithDetails(fmt.Sprintf("body contains unknown field %s", fieldName))
		case errors.As(err, &maxBytesError):
			if maxBytesError.Limit < 1<<20 { // 1MB
				_ = malformedErr.WithDetails(fmt.Sprintf("body must not be larger than %d KB", maxBytesError.Limit/1024))
			} else {
				_ = malformedErr.WithDetails(fmt.Sprintf("body must not be larger than %d MB", maxBytesError.Limit/(1<<20)))
			}
		case errors.As(err, &invalidUnmarshalError):
			_ = malformedErr.WithDetails("body contains invalid JSON")
		default:
			_ = malformedErr.WithDetails("body contains invalid JSON")
		}

		return malformedErr

	}

	// This is to ensure that the body contains only a single JSON value.
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errorx.NewMalformedJSON().WithDetails("body must only contain a single JSON value").WithCause(err, op)
	}

	return nil
}

func ReadUUIDUrlParam(r *http.Request, param string) (uuid.UUID, error) {
	const op = "httpx.ReadUUIDUrlParam"
	idStr := chi.URLParam(r, param)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, errorx.NewInvalidRequest().WithCause(err, op)
	}
	return id, nil
}

func WriteJSON(w http.ResponseWriter, status int, data Envelope, headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	js = append(js, '\n')

	maps.Copy(w.Header(), headers)

	w.Header().Set("Content-Type", "applications/json")
	w.WriteHeader(status)
	_, err = w.Write(js)
	if err != nil {
		return err
	}
	return nil
}

func Success(w http.ResponseWriter, r *http.Request, status int, message Envelope) {
	if message == nil {
		message = make(Envelope, 1)
	}
	message["success"] = true

	err := WriteJSON(w, status, message, nil)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to write success response", "status", status)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
