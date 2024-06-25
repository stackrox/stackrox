package errox

import (
	"net"
	"strconv"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stretchr/testify/assert"
)

func TestWithUserMessage(t *testing.T) {

	tests := map[string]struct {
		err             error
		expectedMessage string
	}{
		"nil base": {
			WithUserMessage(nil, "message"),
			"message",
		},
		"two user messages": {
			WithUserMessage(WithUserMessage(nil, "message"), "second"),
			"second: message",
		},
		"message in between": {
			errors.Wrap(WithUserMessage(errors.New("first"), "message"), "second"),
			"second: message: first",
		},
		"with sentinel": {
			WithUserMessage(NotFound.New("first").New("second"), "message"),
			"message: second",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expectedMessage, test.err.Error())
		})
	}

	assert.ErrorIs(t, WithUserMessage(NotFound, "message"), NotFound)
}

func TestGetUserMessage(t *testing.T) {
	tests := map[string]struct {
		err             error
		expectedMessage string
	}{
		"nil": {
			nil,
			"",
		},
		"message": {
			WithUserMessage(nil, "message"),
			"message",
		},
		"no message": {
			errors.New("no message"),
			"",
		},
		"sensitive with message": {
			errors.WithMessage(WithUserMessage(nil, "message"), "sensitive"),
			"message",
		},
		"message with sensitive": {
			WithUserMessage(errors.WithMessage(nil, "sensitive"), "message"),
			"message",
		},
		"sensitive message sensitive message": {
			err: errors.WithMessage(
				WithUserMessage(
					errors.WithMessage(
						WithUserMessage(nil, "message1"),
						"sensitive1"),
					"message2"),
				"sensitive2"),
			expectedMessage: "message2: message1",
		},
		"with sentinel": {
			WithUserMessage(NotFound.New("first").New("second"), "message"),
			"message: not found",
		},
		"errorlist 0": {
			errorhelpers.NewErrorListWithErrors("start",
				[]error{errors.New("secret")}),
			"start error",
		},
		"errorlist 1": {
			errorhelpers.NewErrorListWithErrors("start",
				[]error{NotFound}),
			"start error: not found",
		},
		"errorlist 2": {
			errorhelpers.NewErrorListWithErrors("start",
				[]error{NotFound, errors.New("secret")}),
			"start error: not found",
		},
		"errorlist 3": {
			errorhelpers.NewErrorListWithErrors("start",
				[]error{NotFound, errors.New("secret"), InvalidArgs}),
			"start errors: [not found, invalid arguments]",
		},
		"errorlist 4": {
			errorhelpers.NewErrorListWithErrors("start", []error{}),
			"",
		},
		"multierror 0": {
			&multierror.Error{},
			"",
		},
		"multierror 1": {
			&multierror.Error{Errors: []error{NotFound}},
			"not found",
		},
		"multierror 2": {
			&multierror.Error{Errors: []error{NotFound, errors.New("secret")}},
			"not found",
		},
		"multierror 3": {
			&multierror.Error{Errors: []error{NotFound, errors.New("secret"), InvalidArgs}},
			"[not found, invalid arguments]",
		},
		"net.AddrError": {
			&net.AddrError{Err: "bad", Addr: "1.2.3.4"},
			"address: bad",
		},
		"net.DNSError": {
			&net.DNSError{Err: "bad", Name: "name", Server: "server"},
			"lookup: bad",
		},
		"net.OpError with unknown error": {
			&net.OpError{Op: "dial", Net: "tcp",
				Err:    errors.New("refused"),
				Source: &net.IPAddr{IP: net.IPv4(1, 2, 3, 4)}},
			"dial tcp",
		},
		"net.OpError with net.DNSError": {
			&net.OpError{Op: "dial", Net: "tcp",
				Err:    &net.DNSError{Err: "bad", Server: "server"},
				Source: &net.IPAddr{IP: net.IPv4(1, 2, 3, 4)}},
			"dial tcp: lookup: bad",
		},
		"strconv": {
			func() error { _, err := strconv.Atoi("abc"); return err }(),
			"error parsing \"abc\"",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expectedMessage, GetUserMessage(test.err))
		})
	}
}
