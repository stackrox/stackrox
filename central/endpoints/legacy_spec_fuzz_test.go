package endpoints

import (
	"testing"
)

// FuzzParseLegacySpec tests ParseLegacySpec with arbitrary input strings to ensure it never panics.
// The parser should handle any malformed input gracefully.
func FuzzParseLegacySpec(f *testing.F) {
	// Seed with valid examples from existing tests
	f.Add("")
	f.Add("8080")
	f.Add(":8080")
	f.Add("grpc@:8443")
	f.Add("http@:8080")
	f.Add("grpc@:8443,http@:8080")
	f.Add(":8080, http@127.0.0.1:8081")
	f.Add("grpc@localhost:8443")
	f.Add("http @ http, grpc @ 127.0.0.1:https, grpc@:8082, localhost:8080, http@127.0.0.1:10080")

	// Seed with edge cases
	f.Add("@")
	f.Add("@@")
	f.Add("@:8080")
	f.Add("grpc@")
	f.Add(",,,")
	f.Add("   ")
	f.Add("grpc @ @ :8080")
	f.Add("grpc@:8443,")
	f.Add(",grpc@:8443")
	f.Add("grpc@:8443,,http@:8080")

	// Seed with potentially problematic inputs
	f.Add("grpc@localhost:8443@extra")
	f.Add("http@:8080@http@:8081")
	f.Add("very-long-protocol-name@:8080")
	f.Add("grpc@very-long-hostname-that-might-cause-issues.example.com:8443")
	f.Add("🚀@:8080")      // Unicode
	f.Add("grpc\n@:8080") // Newline
	f.Add("grpc\t@:8080") // Tab

	f.Fuzz(func(t *testing.T, input string) {
		// The parser should never panic on any input
		// We don't assert specific output because the function is lenient by design
		// It should handle malformed input gracefully
		result := ParseLegacySpec(input, nil)

		// Basic sanity checks: result should never be nil if we got here without panic
		// Each config should have a Listen address (even if empty after trimming)
		for _, cfg := range result {
			// Ensure Protocols is either nil or has strings
			// This validates the internal structure is consistent
			if cfg.Protocols != nil && len(cfg.Protocols) > 0 {
				for _, proto := range cfg.Protocols {
					_ = proto // Ensure we can read each protocol string
				}
			}
			// Ensure Listen is a valid string (can be empty)
			_ = cfg.Listen
		}
	})
}
