package testutils

import (
	"crypto/fips140"
	"testing"
	"testing/cryptotest"
)

// PrepareRandomForSynctest avoids sampling clock jitter inside a synctest bubble.
// Call it only from serial tests that do not depend on cryptographic randomness.
func PrepareRandomForSynctest(t *testing.T) {
	t.Helper()
	// Module v1.0.0 uses OS entropy and does not support SetGlobalRandom.
	if !fips140.Enabled() || fips140.Version() == "v1.0.0" {
		return
	}
	cryptotest.SetGlobalRandom(t, 0)
}
