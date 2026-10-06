package errors

import "errors"

type sentinelMessageError struct {
	sentinel error
	message  string
}

func (e *sentinelMessageError) Error() string { return e.message }

func (e *sentinelMessageError) Is(target error) bool { return target == e.sentinel }

func (e *sentinelMessageError) Unwrap() error { return e.sentinel }

func BadRequest(message string) error {
	return &sentinelMessageError{sentinel: ErrBadRequest, message: message}
}

var (
	ErrUnauthorized = errors.New("unauthorized access")
	ErrForbidden    = errors.New("forbidden resource")
	ErrNotFound     = errors.New("resource not found")
	ErrConflict     = errors.New("resource conflict")
	ErrBadRequest   = errors.New("bad request")
	ErrInternal     = errors.New("internal server error")
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
	ErrInvalidKey   = errors.New("invalid encryption key")
	ErrBadGateway   = errors.New("bad gateway")
)
