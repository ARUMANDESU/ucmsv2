package apperr

import (
	"maps"
	"net/http"
)

type Code string

const (
	CodeInternal         Code = "INTERNAL"
	CodeNotFound         Code = "NOT_FOUND"
	CodeInvalid          Code = "INVALID"
	CodeConflict         Code = "CONFLICT"
	CodeUnauthorized     Code = "UNAUTHORIZED"
	CodeForbidden        Code = "forbidden"
	CodeAlreadyProcessed Code = "ALREADY_PROCESSED"
)

type Error struct {
	Code     Code
	Message  string
	Details  map[string]any
	HTTPCode int // http status code hint
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) WithDetails(details map[string]any) *Error {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}

	maps.Copy(e.Details, details)

	return e
}

func (e *Error) WithCause(cause error) *Error {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}

	e.Details["cause"] = cause.Error()

	return e
}

func (e *Error) HTTPStatusCode() int {
	if e.HTTPCode != 0 {
		return e.HTTPCode
	}

	return HTTPStatusCode(e.Code)
}

func New(code Code, msg string, httpcode int) *Error {
	return &Error{
		Code:     code,
		Message:  msg,
		Details:  nil,
		HTTPCode: httpcode,
	}
}

func NewInternal(msg string) *Error {
	return New(CodeInternal, msg, http.StatusInternalServerError)
}

func NewNotFound(msg string) *Error {
	return New(CodeNotFound, msg, http.StatusNotFound)
}

func NewInvalid(msg string) *Error {
	return New(CodeInvalid, msg, http.StatusBadRequest)
}

func NewConflict(msg string) *Error {
	return New(CodeConflict, msg, http.StatusConflict)
}

func NewUnauthorized(msg string) *Error {
	return New(CodeUnauthorized, msg, http.StatusUnauthorized)
}

func NewForbidden(msg string) *Error {
	return New(CodeForbidden, msg, http.StatusForbidden)
}

func NewAlreadyProcessed(msg string) *Error {
	return New(CodeAlreadyProcessed, msg, http.StatusConflict)
}

func HTTPStatusCode(code Code) int {
	switch code {
	case CodeInternal:
		return http.StatusInternalServerError
	case CodeNotFound:
		return http.StatusNotFound
	case CodeInvalid:
		return http.StatusBadRequest
	case CodeConflict:
		return http.StatusConflict
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeAlreadyProcessed:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError // Default to internal server error for unknown codes
	}
}
