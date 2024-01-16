package notifiers

// NotifierError represents an error that occurs in notifier calls.
type NotifierError struct {
	// msg provides additional information or context about the error.
	msg string

	// err holds the underlying error.
	err error
}

// NewNotifierError creates and returns a new instance of NotifierError.
//
// Parameters:
//   - msg: A string providing additional information or context about the error.
//     This message will be visible by users, and it should not expose any
//     sensitive information.
//   - err: The underlying error that caused the extraction failure.
//
// Returns:
//   - An initialized pointer to an NotifierError struct with the given parameters.
func NewNotifierError(msg string, err error) *NotifierError {
	return &NotifierError{
		msg: msg,
		err: err,
	}
}

func (e *NotifierError) Error() string {
	if e == nil {
		return ""
	}

	return e.msg
}

func (e *NotifierError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.err
}
