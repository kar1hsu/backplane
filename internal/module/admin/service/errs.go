package service

import (
	"errors"

	"gorm.io/gorm"
)

// notFoundOr returns a friendly "not found" error when err is gorm's
// ErrRecordNotFound, otherwise returns the original error — so real DB failures
// (timeouts, connection drops) are not masked as "record not found".
func notFoundOr(err error, notFoundMsg string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New(notFoundMsg)
	}
	return err
}

// forbiddenError marks an action the operator is not allowed to perform, so
// handlers can answer with errcode.ErrForbidden rather than a server error.
type forbiddenError struct{ msg string }

func (e *forbiddenError) Error() string { return e.msg }

func forbidden(msg string) error { return &forbiddenError{msg: msg} }

// IsForbidden reports whether err is a permission rule violation.
func IsForbidden(err error) bool {
	var f *forbiddenError
	return errors.As(err, &f)
}
