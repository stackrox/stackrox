package majorversions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBumpsData(t *testing.T) {
	tests := map[string]struct {
		input   string
		wantErr string
	}{
		"valid": {
			input: `bumps:
  - from: "3.74"
    to: "4.0"`,
		},
		"duplicate to major": {
			input: `bumps:
  - from: "3.74"
    to: "4.0"
  - from: "3.99"
    to: "4.0"`,
			wantErr: "duplicate 'to' major 4",
		},
		"duplicate from major": {
			input: `bumps:
  - from: "3.74"
    to: "4.0"
  - from: "3.99"
    to: "5.0"`,
			wantErr: "duplicate 'from' major 3",
		},
		"invalid from": {
			input: `bumps:
  - from: "bad"
    to: "4.0"`,
			wantErr: "invalid 'from' value",
		},
		"invalid to": {
			input: `bumps:
  - from: "3.74"
    to: "nope"`,
			wantErr: "invalid 'to' value",
		},
		"invalid yaml": {
			input:   `[not valid`,
			wantErr: "yaml:",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := parseBumpsData([]byte(tt.input))
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGetPreviousYStream(t *testing.T) {
	tests := map[string]struct {
		major     int
		minor     int
		wantMajor int
		wantMinor int
		wantErr   string
	}{
		"minor decrement": {
			major: 4, minor: 1,
			wantMajor: 4, wantMinor: 0,
		},
		"ordinary minor": {
			major: 5, minor: 10,
			wantMajor: 5, wantMinor: 9,
		},
		"large minor": {
			major: 45, minor: 67,
			wantMajor: 45, wantMinor: 66,
		},
		"major bump 4.0 -> 3.74": {
			major: 4, minor: 0,
			wantMajor: 3, wantMinor: 74,
		},
		"major bump 5.0 -> 4.11": {
			major: 5, minor: 0,
			wantMajor: 4, wantMinor: 11,
		},
		"unknown major": {
			major: 99, minor: 0,
			wantErr: "don't know the previous Y-Stream for 99.0",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotMajor, gotMinor, err := GetPreviousYStream(tt.major, tt.minor)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantMajor, gotMajor)
			assert.Equal(t, tt.wantMinor, gotMinor)
		})
	}
}
