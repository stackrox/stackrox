package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXyzVersion_ParseFrom(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    XyzVersion
		wantErr bool
	}{
		{
			name:  "simple version",
			input: "3.74.0",
			want:  XyzVersion{X: 3, Y: 74, Z: 0},
		},
		{
			name:  "with patch",
			input: "4.1.2",
			want:  XyzVersion{X: 4, Y: 1, Z: 2},
		},
		{
			name:  "with build suffix",
			input: "3.74.0-123",
			want:  XyzVersion{X: 3, Y: 74, Z: 0},
		},
		{
			name:  "nightly build",
			input: "3.74.x-nightly-20230224",
			want:  XyzVersion{X: 3, Y: 74, Z: 0},
		},
		{
			name:    "invalid format",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseXyzVersion(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestXyzVersion_String(t *testing.T) {
	v := XyzVersion{X: 3, Y: 74, Z: 2}
	assert.Equal(t, "3.74.2", v.String())
}
