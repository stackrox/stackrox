package errox

import (
	"fmt"
	"net"
	"net/url"
	"strings"

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

type multipleErrors interface{ Unwrap() []error }

// wrapErrors is a copy from the standard errors package:
// https://cs.opensource.google/go/go/+/refs/tags/go1.22.6:src/fmt/errors.go;l=67
type wrapErrors struct {
	msg  string
	errs []error
}

func (e *wrapErrors) Error() string {
	return e.msg
}

func (e *wrapErrors) Unwrap() []error {
	return e.errs
}

var _ multipleErrors = (*wrapErrors)(nil)

func concealMultiple(errs multipleErrors) error {
	unwrapped := errs.Unwrap()
	concealed := make([]error, 0, len(unwrapped))
	msg := make([]string, 0, len(unwrapped))
	for _, e := range unwrapped {
		e = ConcealSensitive(e)
		concealed = append(concealed, e)
		msg = append(msg, e.Error())
	}
	return &wrapErrors{
		msg:  fmt.Sprintf("[%s]", strings.Join(msg, ", ")),
		errs: concealed,
	}
}

// ConcealSensitive strips sensitive data from some known error types and
// returns a new error if something has been concealed, and otherwise the
// original error is returned.
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
		// multipleErrors can be returned by fmt.Errorf() with multiple %w.
		if errs, ok := err.(multipleErrors); ok {
			return concealMultiple(errs)
		}
		err = errors.Unwrap(err)
	}
	return original
}
