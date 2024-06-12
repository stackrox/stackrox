package errox

import (
	"net"
	"runtime"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
)

// SensitiveError defines an interface for sensitive errors.
// protect() prevents the Error() method to print sensitive data, and
// unprotect() allows the Error() method, called in the same goroutine, to
// print sensitive data.
type SensitiveError interface {
	error
	protect()
	unprotect()
}

// RoxSensitiveError implements SensitiveError interface.
type RoxSensitiveError struct {
	public    error
	sensitive error

	unprotectedGoRoutineID int
	idMux                  sync.Mutex
}

// Ensure RoxSensitiveError implements errox.SensitiveError.
var _ SensitiveError = (*RoxSensitiveError)(nil)

// MakeSensitive wraps an error with sensitive message, by providing
// a public replacement message, and the API to expose the original one.
func MakeSensitive(public string, err error) *RoxSensitiveError {
	return &RoxSensitiveError{
		public:                 errors.New(public),
		sensitive:              err,
		unprotectedGoRoutineID: -1}
}

// getGoroutineID() returns the number of the current goroutine. It parses the
// first line of the stack, which should look like "goroutine 1 [running]:".
func getGoroutineID() int {
	const n = len("goroutine ")
	var buffer [31]byte
	_ = runtime.Stack(buffer[:], false)
	id, _ := strconv.Atoi(strings.SplitN(string(buffer[n:]), " ", 2)[0])
	return id
}

func (e *RoxSensitiveError) protect() {
	e.unprotectedGoRoutineID = -1
	e.idMux.Unlock()
}

func (e *RoxSensitiveError) unprotect() {
	e.idMux.Lock()
	e.unprotectedGoRoutineID = getGoroutineID()
}

// Unwrap supports the errors.Unwrap() feature.
func (e *RoxSensitiveError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.public
}

// GetSensitiveError returns the full error message with all data from
// occasional sensitive errors exposed.
func GetSensitiveError(err error) string {
	if sensitive := (SensitiveError)(nil); errors.As(err, &sensitive) {
		sensitive.unprotect()
		defer sensitive.protect()
	}
	return err.Error()
}

// isProtected tells whether the error is proteced in the current goroutine.
func (e *RoxSensitiveError) isProtected() bool {
	return e.unprotectedGoRoutineID != getGoroutineID()
}

// Error implements the error interface.
func (e *RoxSensitiveError) Error() string {
	if e == nil {
		return ""
	}
	if !e.isProtected() {
		return GetSensitiveError(e.sensitive)
	}
	// If there is another sensitive error in the chain, add its public message.
	if serr := (SensitiveError)(nil); errors.As(e.sensitive, &serr) {
		return e.public.Error() + ": " + serr.Error()
	}
	return e.public.Error()
}

// ConsealSensitive converts the given error, if it matches one of the known
// error types, to a sensitive error. Otherwise returns the given error.
//
// Example:
//
//	conn, err := connect(secret_ip)
//	err = ConsealSensitive(err)
//	err.Error() // no secret_ip in the text
//	GetSensitiveError(err) // full error text
func ConsealSensitive(err error) error {
	if e := (*net.DNSError)(nil); errors.As(err, &e) {
		return MakeSensitive("lookup: "+e.Err, err)
	}
	if e := (*net.OpError)(nil); errors.As(err, &e) {
		s := e.Op
		if e.Net != "" {
			s += " " + e.Net
		}
		s += ": " + e.Err.Error()
		return MakeSensitive(s, err)
	}
	return err
}
