package errorhelpers

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// ErrorList is a wrapper around many errors
type ErrorList struct {
	start  string
	errors []error
}

// NewErrorList returns a new ErrorList
func NewErrorList(start string) *ErrorList {
	return &ErrorList{
		start: start,
	}
}

// NewErrorListWithErrors returns a new ErrorList with the given errors.
func NewErrorListWithErrors(start string, errors []error) *ErrorList {
	errorList := NewErrorList(start)
	for _, err := range errors {
		errorList.AddError(err)
	}
	return errorList
}

// AddError adds the passed error to the list of errors if it is not nil
func (e *ErrorList) AddError(err error) {
	if err == nil {
		return
	}
	e.errors = append(e.errors, err)
}

// AddErrors adds the non-nil errors in the given slice to the list of errors.
func (e *ErrorList) AddErrors(errs ...error) {
	for _, err := range errs {
		if err == nil {
			continue
		}
		e.errors = append(e.errors, err)
	}
}

// AddWrap is a convenient wrapper around `AddError(errors.Wrap(err, msg))`.
func (e *ErrorList) AddWrap(err error, msg string) {
	e.AddError(errors.Wrap(err, msg))
}

// AddWrapf is a convenient wrapper around `AddError(errors.Wrapf(err, format, args...))`.
func (e *ErrorList) AddWrapf(err error, format string, args ...interface{}) {
	e.AddError(errors.Wrapf(err, format, args...))
}

// AddString adds a string based error to the list
func (e *ErrorList) AddString(err string) {
	e.errors = append(e.errors, errors.New(err))
}

// AddStringf adds a templated string
func (e *ErrorList) AddStringf(t string, args ...interface{}) {
	e.errors = append(e.errors, errors.Errorf(t, args...))
}

// AddStrings adds multiple string based errors to the list.
func (e *ErrorList) AddStrings(errs ...string) {
	for _, err := range errs {
		e.errors = append(e.errors, errors.New(err))
	}
}

// ToError returns an error if there were errors added or nil
func (e *ErrorList) ToError() error {
	if len(e.errors) == 0 {
		return nil
	}
	return e
}

// Error implements the error interface
func (e *ErrorList) Error() string {
	return e.String()
}

// String converts the list to a string, returning empty if no errors were added.
func (e *ErrorList) String() string {
	switch len(e.errors) {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf("%s error: %s", e.start, e.errors[0])
	default:
		return fmt.Sprintf("%s errors: [%s]", e.start, strings.Join(e.ErrorStrings(), ", "))
	}
}

// ErrorStrings returns all the error strings in this ErrorList as a slice, ignoring the start string.
func (e *ErrorList) ErrorStrings() []string {
	errors := make([]string, 0, len(e.errors))
	for _, err := range e.errors {
		errors = append(errors, err.Error())
	}
	return errors
}

// Errors returns the underlying errors in the error list
func (e *ErrorList) Errors() []error {
	return e.errors
}

// Empty returns whether the list of error strings is empty.
func (e *ErrorList) Empty() bool {
	return len(e.errors) == 0
}
