package errox

import (
	"net"

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

// ConcealSensitive strips sensitive data from some known error types and
// returns a new error.
func ConcealSensitive(err error) error {
	if err == nil {
		return nil
	}
	if e := (*net.AddrError)(nil); errors.As(err, &e) {
		return errors.New("address error: " + e.Err)
	}
	if e := (*net.DNSError)(nil); errors.As(err, &e) {
		return errors.New("lookup error: " + e.Err)
	}
	if e := (*net.OpError)(nil); errors.As(err, &e) {
		s := e.Op
		if e.Net != "" {
			s += " " + e.Net
		}
		s += ": " + e.Err.Error()
		return errors.New(s)
	}
	return err
}
