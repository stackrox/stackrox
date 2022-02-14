package errox

import "fmt"

type parameterizedError func(args ...interface{}) *roxError

// New creates an error based on another error, e.g., an existing sentinel
// error, but with the personalized error message. Essentially, it allows to
// preserve the error base error in the chain but hide its message.
//
// Example:
//     myPackageError := errox.New(errox.NotFound, "file not found")
//     myPackageError.Error() == "file not found" // true
//     errors.Is(myPackageError, errox.NotFound)  // true
func New(base error, message string) *roxError {
	return &roxError{message, base}
}

// Newf helps creating a parameterized error, based on another error, e.g.,
// an existing sentinel error. The provided format specifier defines the
// expected arguments of the returned function.
//
// Example:
//     myPackageError := errox.Newf(errox.NotFound, "file '%s' not found")
//     myPackageError("sh").Error() == "file 'sh' not found" // true
//     errors.Is(myPackageError, errox.NotFound)             // true
func Newf(base error, format string) parameterizedError {
	return func(args ...interface{}) *roxError {
		return New(base, fmt.Sprintf(format, args...))
	}
}

// Explain adds an explanation to the error and returns a new error. It appends
// the provided explanation to the original error message.
//
// Example:
//     return Explain(err, "all workers are busy")
func Explain(err error, explanation string) error {
	return fmt.Errorf("%w, %s", err, explanation)
}

// New is a syntactic sugar for New(e, message).
func (e *roxError) New(message string) *roxError {
	return New(e, message)
}

// Newf is a syntactic sugar for Newf(e, message).
func (e *roxError) Newf(format string) parameterizedError {
	return Newf(e, format)
}

// Explain is a syntactic sugar for Explain(e, message).
func (e *roxError) Explain(explanation string) error {
	return Explain(e, explanation)
}
