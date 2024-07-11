package netutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLocalHost_True(t *testing.T) {
	t.Parallel()

	hosts := []string{
		"::1",
		"127.0.0.1",
		"127.100.100.1",
		"::ffff:7f00:0001",
		"localhost",
	}

	for _, host := range hosts {
		assert.Truef(t, IsLocalHost(host), "Expected host %q to be recognized as local", host)
	}
}

func TestIsLocalHost_False(t *testing.T) {
	t.Parallel()

	hosts := []string{
		"::ffff:7e00:0001",
		"128.0.0.1",
		"192.168.0.1",
		"local",
		"example.com",
	}

	for _, host := range hosts {
		assert.Falsef(t, IsLocalHost(host), "Expected host %q to be recognized as non-local", host)
	}
}
