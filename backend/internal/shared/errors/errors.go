package errors

import "errors"

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
)
