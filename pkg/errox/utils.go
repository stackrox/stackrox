package errox

import (
	"fmt"
	"net"
	"net/url"

	"github.com/pkg/errors"
)

// IsAny returns a bool if it matches any of the target errors
// This helps consolidate code from
// errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.ErrClosedPipe)
// to errors.IsAny(err, io.EOF. io.ErrUnexpectedEOF, io.ErrClosedPipe)
func IsAny(err error, targets ...error) bool {
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

func x[T error](err error) T {
	var e T
	if errors.Is(err, e) {
		return e
	}
	return e
}

// ConcealSensitive strips sensitive data from some known error types and
// returns a new error.
func ConcealSensitive(err error) error {
	original := err
	for err != nil {
		// Here are reimplementations of the *.Error() methods of the original
		// errors, without including potentially sensitive data to the message.
		switch e := err.(type) {
		case *net.AddrError:
			return errors.New("address error: " + e.Err)
		case *net.DNSError:
			return errors.New("lookup error: " + e.Err)
		case *net.OpError:
			s := e.Op
			if e.Net != "" {
				s += " " + e.Net
			}
			if e.Err != nil {
				s += ": " + ConcealSensitive(e.Err).Error()
			}
			return errors.New(s)
		case *url.Error:
			return errors.New(fmt.Sprintf("%s %q: %s", e.Op, e.URL, ConcealSensitive(e.Err)))
		}
		err = errors.Unwrap(err)
	}
	return original
}
