package httputil

// HTTPStatus is an interface for statuses that can be returned from an HTTP handler.
type HTTPStatus interface {
	Message() string
	HTTPStatusCode() int
}

type httpStatus struct {
	code    int
	message string
}

func (s httpStatus) Message() string {
	return s.message
}

func (s httpStatus) HTTPStatusCode() int {
	return s.code
}

// NewStatus returns a new http status object with the given code and message.
func NewStatus(code int, message string) HTTPStatus {
	return httpStatus{code: code, message: message}
}
