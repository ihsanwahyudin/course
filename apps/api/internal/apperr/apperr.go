// Package apperr is the client-facing error shape shared by every service.
package apperr

import "fmt"

// Error is a failure with an HTTP status and a stable code clients can act on.
type Error struct {
	Status    int    `json:"-"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func New(status int, code string, retryable bool, format string, args ...any) *Error {
	return &Error{Status: status, Code: code, Message: fmt.Sprintf(format, args...), Retryable: retryable}
}

var Unauthorized = &Error{Status: 401, Code: "unauthorized", Message: "Please log in to continue."}
