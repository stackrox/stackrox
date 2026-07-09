package clairv4

import (
	"crypto/fips140"
	stdos "os"
	"testing"
)

// quay/claircore uses MD5 for non-cryptographic package integrity checks.
func TestMain(m *testing.M) {
	var code int
	fips140.WithoutEnforcement(func() {
		code = m.Run()
	})
	stdos.Exit(code)
}
