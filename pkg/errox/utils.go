package errox

import "github.com/pkg/errors"

// As simplifies error type tests by hiding the temporary variable declaration.
// Example:
//
//	var he HTTPStatus
//	if errors.As(err, &he) {
//		return he.HTTPStatusCode()
//	}
//
// can be rewritten as:
//
//	if he := errox.As[HTTPStatus](err); he != nil {
//		return he.HTTPStatusCode()
//	}
func As[T error](e error) T {
	var ptr T
	errors.As(e, &ptr)
	return ptr
}
