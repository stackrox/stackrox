package errox

import (
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestConcealSensitive(t *testing.T) {

	tests := map[string]struct {
		err      error
		expected string
	}{
		"simple": {
			errors.New("simple"),
			"simple",
		},
		"sentinel": {
			NotAuthorized,
			"not authorized",
		},
		"net.AddrError": {
			&net.AddrError{Err: "invalid IP address", Addr: "secret"},
			"address error: invalid IP address",
		},
		"net.DNSError": {
			&net.DNSError{Err: "unknown network", Name: "secret", Server: "server"},
			"lookup error: unknown network",
		},
		"net.OpError": {
			&net.OpError{Op: "operation", Net: "tcp", Err: NotFound, Addr: &net.IPAddr{IP: net.IPv4(1, 2, 3, 4)}},
			"operation tcp: not found",
		},
		"url.Error": {
			&url.Error{Err: &net.OpError{Op: "operation", Err: NotFound, Addr: &net.IPAddr{IP: net.IPv4(1, 2, 3, 4)}}, Op: "Get", URL: "URL"},
			// The url.Error details are not dropped, because url.Error is a
			// known error type. The wrapped net.OpError is concealed.
			"Get \"URL\": operation: not found",
		},
		"wrapped": {
			errors.WithMessage(&net.AddrError{Err: "invalid IP address", Addr: "secret"}, "dropped message"),
			// The "dropped message" is dropped, because the type of the wrapper
			// is unknown.
			"address error: invalid IP address",
		},
		"%w": {
			fmt.Errorf("%w", &net.AddrError{Err: "invalid IP address", Addr: "secret"}),
			"address error: invalid IP address",
		},
		"%w %w %w": {
			fmt.Errorf("dropped %w wrapping %w message %w",
				&net.AddrError{Err: "invalid IP address", Addr: "secret"},
				&net.OpError{Op: "operation", Err: NotFound, Addr: &net.IPAddr{IP: net.IPv4(1, 2, 3, 4)}},
				&net.DNSError{Err: "unknown network", Name: "secret", Server: "server"}),
			"[address error: invalid IP address, operation: not found, lookup error: unknown network]",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, ConcealSensitive(test.err).Error())
		})
	}
	assert.Nil(t, ConcealSensitive(nil))
}
