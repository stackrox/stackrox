package diff

import (
	"crypto/fips140"
	"os"
	"testing"
)

// np-guard/netpol-analyzer uses SHA-1 for non-cryptographic label hashing.
func TestMain(m *testing.M) {
	var code int
	fips140.WithoutEnforcement(func() {
		code = m.Run()
	})
	os.Exit(code)
}
