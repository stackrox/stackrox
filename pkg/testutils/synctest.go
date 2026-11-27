package testutils

import (
	"crypto/fips140"
	"testing"
	"testing/cryptotest"
)

// PrepareRandomForSynctest avoids sampling clock jitter inside a synctest bubble.
// Call it only from serial tests that do not depend on cryptographic randomness.
//
// Deprecated: This helper is a temporary workaround for Go 1.26.
// The upstream fix uses the system monotonic clock inside synctest bubbles:
// https://github.com/golang/go/commit/a81b20a431ce41417f7758929a2962d95a6a9453.
//
// TODO: Once Go 1.27 is the minimum supported version, remove this helper and
// its calls after verifying the affected tests pass with FIPS enabled without it.
// See https://go.dev/issue/78557.
func PrepareRandomForSynctest(t *testing.T) {
	t.Helper()
	if !fips140.Enabled() {
		t.Log("Using the default random source because FIPS mode is disabled")
		return
	}
	// Module v1.0.0 uses OS entropy and does not support SetGlobalRandom.
	if fips140.Version() == "v1.0.0" {
		t.Log("Using OS entropy with FIPS module v1.0.0")
		return
	}
	cryptotest.SetGlobalRandom(t, 0)
}
