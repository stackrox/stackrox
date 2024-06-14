package errox

import (
	"math"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
)

const unsetID uint64 = math.MaxUint64

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

	unprotectedGoRoutineID atomic.Uint64
	protectionMux          sync.Mutex
}

// Ensure RoxSensitiveError implements errox.SensitiveError.
var _ SensitiveError = (*RoxSensitiveError)(nil)

// MakeSensitive wraps an error with sensitive message, by providing
// a public replacement message, and the API to expose the original one.
func MakeSensitive(public string, err error) *RoxSensitiveError {
	result := &RoxSensitiveError{
		public:    errors.New(public),
		sensitive: err}
	result.unprotectedGoRoutineID.Store(unsetID)
	return result
}

// getGoroutineID() returns the number of the current goroutine. It parses the
// first line of the stack, which should look like "goroutine 1 [running]:".
func getGoroutineID() int {
	const goroutine = len("goroutine ")
	const maxLenght = len("goroutine 18446744073709551615 ")
	var buffer [maxLenght]byte
	_ = runtime.Stack(buffer[:], false)
	id, _ := strconv.Atoi(strings.SplitN(string(buffer[goroutine:]), " ", 2)[0])
	return id
}

// Disable Error() to write sensitive data.
func (e *RoxSensitiveError) protect() {
	e.unprotectedGoRoutineID.Store(unsetID)
	e.protectionMux.Unlock()
}

// Enable Error() to write sensitive data in the current goroutine.
func (e *RoxSensitiveError) unprotect() {
	// Lock during the period without protection.
	e.protectionMux.Lock()
	e.unprotectedGoRoutineID.Store(uint64(getGoroutineID()))
}

// Unwrap supports the errors.Unwrap() feature.
func (e *RoxSensitiveError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.public
}

// UnconcealSensitive returns the full error message with all data from
// occasional sensitive errors exposed.
func UnconcealSensitive(err error) string {
	if sensitive := (SensitiveError)(nil); errors.As(err, &sensitive) {
		sensitive.unprotect()
		defer sensitive.protect()
	}
	return err.Error()
}

// isProtected tells whether the error is proteced in the current goroutine.
func (e *RoxSensitiveError) isProtected() bool {
	return e.unprotectedGoRoutineID.Load() != uint64(getGoroutineID())
}

// Error implements the error interface.
func (e *RoxSensitiveError) Error() string {
	if e == nil {
		return ""
	}
	if !e.isProtected() {
		return UnconcealSensitive(e.sensitive)
	}
	// If there is another sensitive error in the chain, add its public message.
	if serr := (SensitiveError)(nil); errors.As(e.sensitive, &serr) {
		return e.public.Error() + ": " + serr.Error()
	}
	return e.public.Error()
}

// ConcealSensitive converts the given error, if it matches one of the known
// error types, to a sensitive error. Otherwise returns the given error.
//
// Example:
//
//	conn, err := connect(secret_ip)
//	err = ConcealSensitive(err)
//	err.Error() // no secret_ip in the text
//	GetSensitiveError(err) // full error text
func ConcealSensitive(err error) error {
	if err == nil {
		return nil
	}
	if e := (*net.AddrError)(nil); errors.As(err, &e) {
		return MakeSensitive("address: "+e.Err, err)
	}
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
