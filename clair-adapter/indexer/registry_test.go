package indexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseImageRef(t *testing.T) {
	tests := map[string]struct {
		input   string
		wantErr bool
	}{
		"docker hub with tag": {
			input: "docker.io/library/bash:5.1",
		},
		"with https scheme": {
			input: "https://registry.example.com/repo/image:tag",
		},
		"with http scheme": {
			input: "http://registry.example.com/repo/image:tag",
		},
		"digest reference": {
			input: "docker.io/library/bash@sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abcd",
		},
		"empty": {
			input:   "",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ref, err := parseImageRef(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, ref)
		})
	}
}
