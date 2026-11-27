// Package fips enforces the runtime cryptographic mode required by our binaries.
package fips

import (
	"crypto/fips140"
	"fmt"
	"os"
)

// RequireEnabled exits with an error if FIPS 140-3 mode is disabled. Call it at
// the start of main: build defaults can be overridden by GODEBUG at startup.
func RequireEnabled() {
	if !fips140.Enabled() {
		fmt.Fprintln(os.Stderr, "FIPS 140-3 mode is required; do not disable it with GODEBUG=fips140=off")
		os.Exit(1)
	}
}
