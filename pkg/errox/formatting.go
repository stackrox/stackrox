package errox

import (
	"strings"
	"text/template"
)

// New creates an error based on the existing roxError, but with the
// personalized error message. Essentially, it allows for preserving the error
// base error in the chain but hide its message.
//
// Example:
//     var ErrRecordNotFound := errox.NotFound.New("record not found")
//     ErrRecordNotFound.Error() == "record not found" // true
//     errors.Is(ErrRecordNotFound, errox.NotFound)    // true
func (e *roxError) New(message string) *roxError {
	return &roxError{message, e}
}

// Template helps creating a parameterized error message. The template text
// defines the expected argument of the returned function.
// Example:
//     var ErrFileNotFound = errox.NotFound.Template("file '{{.}}' not found")
//     ErrFileNotFound("sh").Error() == "file 'sh' not found" // true
//     errors.Is(ErrFileNotFound, errox.NotFound)             // true
func (e *roxError) Template(text string) func(arg interface{}) *roxError {
	t := template.Must(template.New(text).Parse(text))
	return func(arg interface{}) *roxError {
		b := strings.Builder{}
		if err := t.Execute(&b, arg); err != nil {
			return e
		}
		return e.New(b.String())
	}
}

var causeTemplate = template.Must(template.New("ErrorCause").Parse(
	"{{.Error}}: {{.Cause}}"))

// CausedBy adds a cause to the roxError. The resulting message is a combination
// of the rox error and the cause following a colon.
//
// Example:
//     return errox.InvalidArgument.CausedBy(err)
// or
//     return errox.InvalidArgument.CausedBy("unknown parameter")
func (e *roxError) CausedBy(cause interface{}) error {
	var b strings.Builder
	if err := causeTemplate.Execute(&b, struct {
		Error *roxError
		Cause interface{}
	}{
		Error: e,
		Cause: cause,
	}); err != nil {
		return e
	}
	return e.New(b.String())
}
