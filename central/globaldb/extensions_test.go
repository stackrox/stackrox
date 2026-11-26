package globaldb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNeedsExtensionUpdate(t *testing.T) {
	tests := map[string]struct {
		installedVersion string
		availableVersion string
		expected         bool
	}{
		"matching versions": {
			installedVersion: "1.10",
			availableVersion: "1.10",
			expected:         false,
		},
		"installed version is outdated": {
			installedVersion: "1.10",
			availableVersion: "1.11",
			expected:         true,
		},
		"numeric ordering differs from lexical ordering": {
			installedVersion: "1.9",
			availableVersion: "1.10",
			expected:         true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.expected,
				needsExtensionUpdate(test.installedVersion, test.availableVersion))
		})
	}
}
