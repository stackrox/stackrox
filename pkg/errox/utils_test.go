package errox

import (
	"net"
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
			&net.OpError{Op: "operation", Err: NotFound, Addr: &net.IPAddr{IP: net.IPv4(1, 2, 3, 4)}},
			"operation: not found",
		},
		"wrapped": {
			errors.WithMessage(&net.AddrError{Err: "invalid IP address", Addr: "secret"}, "dropped message"),
			"address error: invalid IP address",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, ConcealSensitive(test.err).Error())
		})
	}
	assert.Nil(t, ConcealSensitive(nil))
}
