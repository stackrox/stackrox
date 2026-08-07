package exitcode

// Error wraps an exit code so commands can signal specific process exit codes
// without calling os.Exit directly. When the err field is nil, Error() returns
// an empty string — this is the "silent exit code" case where the command has
// already printed all its output.
type Error struct {
	code int
	err  error
}

// New creates an Error with the given exit code.
func New(code int, err error) *Error {
	return &Error{code: code, err: err}
}

// Code returns the exit code.
func (e *Error) Code() int {
	return e.code
}

func (e *Error) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return ""
}

// Unwrap supports errors.Is / errors.As on the wrapped error.
func (e *Error) Unwrap() error {
	return e.err
}
