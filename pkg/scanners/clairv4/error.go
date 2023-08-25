package clairv4

import "fmt"

var _ error = (*unexpectedStatusCode)(nil)

type unexpectedStatusCode struct {
	statusCode int
}

func newUnexpectedStatusCodeError(statusCode int) *unexpectedStatusCode {
	return &unexpectedStatusCode{statusCode: statusCode}
}

func (u *unexpectedStatusCode) Error() string {
	return fmt.Sprintf("received unexpected status code from Clair v4: %d", u.statusCode)
}

func isUnexpectedStatusCodeError(err error) bool {
	_, ok := err.(*unexpectedStatusCode)
	return ok
}
