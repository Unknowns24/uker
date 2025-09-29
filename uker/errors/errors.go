package errors

import "fmt"

// Code identifies the error type.
type Code string

// Error represents a domain error with a code and an underlying error.
type Error struct {
	Code    Code
	Message string
	Err     error
}

// New creates an Error with the provided code and message.
func New(code Code, message string) Error {
	return Error{Code: code, Message: message}
}

// Wrap adds context to an existing error while keeping the code intact.
func Wrap(code Code, message string, err error) Error {
	return Error{Code: code, Message: message, Err: err}
}

// Error implements the error interface.
func (e Error) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
}

// Unwrap exposes the wrapped error (if any).
func (e Error) Unwrap() error {
	return e.Err
}
