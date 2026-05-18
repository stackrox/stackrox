package datastore

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestEncodeIssuer(t *testing.T) {
	tests := map[string]struct {
		config   *storage.AuthMachineToMachineConfig
		expected string
	}{
		"DECLARATIVE origin with issuer": {
			config: &storage.AuthMachineToMachineConfig{
				Issuer: "https://example.com",
				Traits: &storage.Traits{
					Origin: storage.Traits_DECLARATIVE,
				},
			},
			expected: "DECLARATIVE|https://example.com",
		},
		"IMPERATIVE origin with issuer": {
			config: &storage.AuthMachineToMachineConfig{
				Issuer: "https://token.actions.githubusercontent.com",
				Traits: &storage.Traits{
					Origin: storage.Traits_IMPERATIVE,
				},
			},
			expected: "IMPERATIVE|https://token.actions.githubusercontent.com",
		},
		"DEFAULT origin with issuer": {
			config: &storage.AuthMachineToMachineConfig{
				Issuer: "https://kubernetes.default.svc",
				Traits: &storage.Traits{
					Origin: storage.Traits_DEFAULT,
				},
			},
			expected: "DEFAULT|https://kubernetes.default.svc",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			encoded := encodeIssuer(tc.config)
			assert.Equal(t, tc.expected, encoded)
		})
	}
}

func TestDecodeIssuer(t *testing.T) {
	tests := map[string]struct {
		encoded  string
		expected string
	}{
		"DECLARATIVE encoded issuer": {
			encoded:  "DECLARATIVE|https://example.com",
			expected: "https://example.com",
		},
		"IMPERATIVE encoded issuer": {
			encoded:  "IMPERATIVE|https://token.actions.githubusercontent.com",
			expected: "https://token.actions.githubusercontent.com",
		},
		"DEFAULT encoded issuer": {
			encoded:  "DEFAULT|https://kubernetes.default.svc",
			expected: "https://kubernetes.default.svc",
		},
		"unencoded issuer (backwards compatibility)": {
			encoded:  "https://plain.issuer.com",
			expected: "https://plain.issuer.com",
		},
		"issuer with pipe in URL": {
			encoded:  "DECLARATIVE|https://example.com/path|with|pipes",
			expected: "https://example.com/path|with|pipes",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			decoded := decodeIssuer(tc.encoded)
			assert.Equal(t, tc.expected, decoded)
		})
	}
}

func TestWithEncodedIssuer(t *testing.T) {
	tests := map[string]struct {
		config         *storage.AuthMachineToMachineConfig
		expectedIssuer string
	}{
		"DECLARATIVE origin preserves original config": {
			config: &storage.AuthMachineToMachineConfig{
				Id:     "test-id",
				Issuer: "https://example.com",
				Traits: &storage.Traits{
					Origin: storage.Traits_DECLARATIVE,
				},
				Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
					{Key: "sub", ValueExpression: "test", Role: "Admin"},
				},
			},
			expectedIssuer: "DECLARATIVE|https://example.com",
		},
		"original config is not modified": {
			config: &storage.AuthMachineToMachineConfig{
				Id:     "test-id-2",
				Issuer: "https://original.com",
				Traits: &storage.Traits{
					Origin: storage.Traits_IMPERATIVE,
				},
			},
			expectedIssuer: "IMPERATIVE|https://original.com",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			originalIssuer := tc.config.Issuer
			encoded := withEncodedIssuer(tc.config)

			// Verify the encoded config has the encoded issuer
			assert.Equal(t, tc.expectedIssuer, encoded.Issuer)

			// Verify the original config is unchanged
			assert.Equal(t, originalIssuer, tc.config.Issuer)

			// Verify other fields are preserved
			assert.Equal(t, tc.config.Id, encoded.Id)
			assert.Equal(t, tc.config.Traits.Origin, encoded.Traits.Origin)
			assert.Equal(t, len(tc.config.Mappings), len(encoded.Mappings))
		})
	}
}
