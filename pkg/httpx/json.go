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
)

type Envelope map[string]any

const maxRequestBodySize = 10 << 20 // 10MB

func ReadJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(v)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("badly-formed JSON (at character %d): %w", syntaxError.Offset, err)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return fmt.Errorf("body contains badly-formed JSON: %w", err)
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains invalid JSON (at character %d): %w", unmarshalTypeError.Offset, err)
			}
			return fmt.Errorf("body contains invalid JSON (at character %d): %w", unmarshalTypeError.Offset, err)
		case errors.Is(err, io.EOF):
			return fmt.Errorf("body must not be empty: %w", err)
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown field %s: %w", fieldName, err)
		case errors.As(err, &maxBytesError):
			if maxBytesError.Limit < 1<<20 { // 1MB
				return fmt.Errorf("body must not be larger than %d KB: %w", maxBytesError.Limit/1024, err)
			}
			return fmt.Errorf("body must not be larger than %d MB: %w", maxBytesError.Limit/(1<<20), err)
		case errors.As(err, &invalidUnmarshalError):
			return fmt.Errorf("body contains invalid JSON: %w", err)
		default:
			return fmt.Errorf("body contains invalid JSON: %w", err)
		}

	}

	// This is to ensure that the body contains only a single JSON value.
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return fmt.Errorf("body must only contain a single JSON value: %w", err)
	}

	return nil
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
