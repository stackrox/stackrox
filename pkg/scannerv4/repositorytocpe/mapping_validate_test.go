package repositorytocpe

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMapping(t *testing.T) {
	tests := map[string]struct {
		content []byte
		wantErr bool
	}{
		"valid minimal JSON accepted": {
			content: []byte(`{"data":{}}`),
			wantErr: false,
		},
		"malformed JSON rejected": {
			content: []byte(`{"data":`),
			wantErr: true,
		},
		"oversize content rejected": {
			content: bytes.Repeat([]byte("a"), MaxMappingBytes+1),
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := ValidateMapping(tt.content)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestHashMapping(t *testing.T) {
	tests := map[string]struct {
		content  []byte
		wantHash string
	}{
		"stable hash for fixed bytes": {
			content:  []byte(`{"data":{"repo-1":{"cpes":["cpe:/o:redhat:rhel:8"]}}}`),
			wantHash: "e34b7b416f5e54e8",
		},
		"empty content hash is still 16 hex chars": {
			content:  []byte(""),
			wantHash: "ef46db3751d8e999",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			hash := HashMapping(tt.content)
			assert.Len(t, hash, 16)
			assert.Equal(t, tt.wantHash, hash)
		})
	}
}
