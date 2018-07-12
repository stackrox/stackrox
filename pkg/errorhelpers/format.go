package errorhelpers

import (
	"fmt"
	"strings"
)

// ErrorList is a wrapper around many errors
type ErrorList struct {
	start  string
	errors []string
}

// NewErrorList returns a new ErrorList
func NewErrorList(start string) *ErrorList {
	return &ErrorList{
		start: start,
	}
}

// AddError adds the passes error to the list of errors if it is not nil
func (e *ErrorList) AddError(err error) {
	if err == nil {
		return
	}
	e.errors = append(e.errors, err.Error())
}

// AddString adds a string based error to the list
func (e *ErrorList) AddString(err string) {
	e.errors = append(e.errors, err)
}

// ToError returns an error if there were errors added or nil
func (e *ErrorList) ToError() error {
	if len(e.errors) > 0 {
		return fmt.Errorf("%s errors: [%s]", e.start, strings.Join(e.errors, ", "))
	}
	return nil
}
