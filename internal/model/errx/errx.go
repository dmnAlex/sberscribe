package errx

import "errors"

var (
	ErrBadRequest       = errors.New("bad request")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrNotFound         = errors.New("not found")
	ErrMethodNotAllowed = errors.New("method not allowed")
	ErrAlreadyExists    = errors.New("already exists")
	ErrConflict         = errors.New("conflict")
	ErrUnprocessable    = errors.New("unprocessable")
	ErrInternalError    = errors.New("internal server error")
)
