package nvd

import (
	"crypto/fips140"
	"os"
	"testing"
)

// quay/claircore uses MD5 for non-cryptographic package integrity checks.
func TestMain(m *testing.M) {
	var code int
	fips140.WithoutEnforcement(func() {
		code = m.Run()
	})
	os.Exit(code)
}
