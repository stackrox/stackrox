package errox

// Stub package for testing erroxwrap analyzer

type RoxError struct {
	message string
}

func (e *RoxError) Error() string {
	return e.message
}

func (e *RoxError) New(message string) *RoxError {
	return &RoxError{message: message}
}

func (e *RoxError) Newf(format string, args ...interface{}) *RoxError {
	return &RoxError{message: format}
}

func (e *RoxError) CausedBy(cause interface{}) error {
	return e
}

func (e *RoxError) CausedByf(format string, args ...interface{}) error {
	return e
}

// Sentinel errors
var (
	NotFound                  = &RoxError{message: "not found"}
	AlreadyExists             = &RoxError{message: "already exists"}
	InvalidArgs               = &RoxError{message: "invalid arguments"}
	ReferencedByAnotherObject = &RoxError{message: "referenced by another object"}
	InvariantViolation        = &RoxError{message: "invariant violation"}
	NoCredentials             = &RoxError{message: "credentials not found"}
	NotAuthorized             = &RoxError{message: "not authorized"}
	NoAuthzConfigured         = &RoxError{message: "service authorization is misconfigured"}
	ServerError               = &RoxError{message: "server error"}
	ResourceExhausted         = &RoxError{message: "resource exhausted"}
	NotImplemented            = &RoxError{message: "not implemented"}
	ReferencedObjectNotFound  = &RoxError{message: "referenced object not found"}
)
