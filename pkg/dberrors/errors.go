package dberrors

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/pkg/errox"
)

// An ErrNotFound indicates that the desired object could not be located.
type errNotFound struct {
	Type string
	ID   string
}

// New returns a new error associated with the provided type and id.
//
// TODO: Consider to replace with:
//   return errors.WithMessage(errorhelpers.ErrNotFound, fmt.Sprintf("%s '%s'", t, id))
func New(t, id string) errox.RoxError {
	return &errNotFound{t, id}
}

// Code returns Rox error code. Implements RoxError interface.
func (e *errNotFound) Code() errox.Code {
	return errox.CodeNotFound
}

// Namespace returns Rox error namespace. Implements RoxError interface.
func (e *errNotFound) Namespace() string {
	return "db"
}

// Error returns error message. Implements error interface.
func (e *errNotFound) Error() string {
	sb := strings.Builder{}
	if e.Type != "" {
		_, _ = sb.WriteString(fmt.Sprintf("%s ", e.Type))
	}
	if e.ID != "" {
		_, _ = sb.WriteString(fmt.Sprintf("'%s' ", e.ID))
	}
	_, _ = sb.WriteString("not found")
	return sb.String()
}

// IsNotFound returns whether a given error is an instance of errNotFound.
func IsNotFound(err error) bool {
	_, ok := err.(*errNotFound)
	return ok
}
