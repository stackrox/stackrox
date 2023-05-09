package httputil

// Is2xxStatusCode checks if the given response code is a 2xx HTTP status code.
func Is2xxStatusCode(code int) bool {
	return code/100 == 2
}

// Is2xxOr3xxStatusCode checks if the given response code is a 2xx or 3xx HTTP status code.
func Is2xxOr3xxStatusCode(code int) bool {
	return Is2xxStatusCode(code) || code/100 == 3
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
