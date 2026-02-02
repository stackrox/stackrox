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
			name:  "version with v prefix",
			input: "v3.74.0",
			want:  XyzVersion{X: 3, Y: 74, Z: 0},
		},
		{
			name:  "multi-digit components",
			input: "123.456.789",
			want:  XyzVersion{X: 123, Y: 456, Z: 789},
		},
		{
			name:  "zero version",
			input: "0.0.0",
			want:  XyzVersion{X: 0, Y: 0, Z: 0},
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

func TestXyzVersion_Compare(t *testing.T) {
	tests := []struct {
		name     string
		v1       XyzVersion
		v2       XyzVersion
		expected int
	}{
		// Equal versions
		{
			name:     "equal versions",
			v1:       XyzVersion{X: 3, Y: 74, Z: 0},
			v2:       XyzVersion{X: 3, Y: 74, Z: 0},
			expected: 0,
		},
		{
			name:     "equal multi-digit versions",
			v1:       XyzVersion{X: 123, Y: 456, Z: 789},
			v2:       XyzVersion{X: 123, Y: 456, Z: 789},
			expected: 0,
		},
		{
			name:     "equal zero versions",
			v1:       XyzVersion{X: 0, Y: 0, Z: 0},
			v2:       XyzVersion{X: 0, Y: 0, Z: 0},
			expected: 0,
		},

		// Major version comparisons
		{
			name:     "major version less",
			v1:       XyzVersion{X: 2, Y: 74, Z: 0},
			v2:       XyzVersion{X: 3, Y: 74, Z: 0},
			expected: -1,
		},
		{
			name:     "major version greater",
			v1:       XyzVersion{X: 4, Y: 74, Z: 0},
			v2:       XyzVersion{X: 3, Y: 74, Z: 0},
			expected: 1,
		},
		{
			name:     "major version less overrides minor/patch",
			v1:       XyzVersion{X: 2, Y: 99, Z: 99},
			v2:       XyzVersion{X: 3, Y: 0, Z: 0},
			expected: -1,
		},

		// Minor version comparisons
		{
			name:     "minor version less",
			v1:       XyzVersion{X: 3, Y: 73, Z: 0},
			v2:       XyzVersion{X: 3, Y: 74, Z: 0},
			expected: -1,
		},
		{
			name:     "minor version greater",
			v1:       XyzVersion{X: 3, Y: 75, Z: 0},
			v2:       XyzVersion{X: 3, Y: 74, Z: 0},
			expected: 1,
		},
		{
			name:     "minor version less overrides patch",
			v1:       XyzVersion{X: 3, Y: 73, Z: 99},
			v2:       XyzVersion{X: 3, Y: 74, Z: 0},
			expected: -1,
		},

		// Patch version comparisons
		{
			name:     "patch version less",
			v1:       XyzVersion{X: 3, Y: 74, Z: 1},
			v2:       XyzVersion{X: 3, Y: 74, Z: 2},
			expected: -1,
		},
		{
			name:     "patch version greater",
			v1:       XyzVersion{X: 3, Y: 74, Z: 3},
			v2:       XyzVersion{X: 3, Y: 74, Z: 2},
			expected: 1,
		},
		{
			name:     "patch version zero vs non-zero",
			v1:       XyzVersion{X: 3, Y: 74, Z: 0},
			v2:       XyzVersion{X: 3, Y: 74, Z: 1},
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.v1.Compare(tt.v2)
			assert.Equal(t, tt.expected, result,
				"Compare(%s, %s) = %d, want %d",
				tt.v1.String(), tt.v2.String(), result, tt.expected)
		})
	}
}

func TestGetPreviousYStream(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
		wantErr bool
	}{
		{
			name:    "minor version decrement",
			version: "3.74.0",
			want:    "3.73.0",
		},
		{
			name:    "minor version decrement with patch",
			version: "3.74.3",
			want:    "3.73.0",
		},
		{
			name:    "major version 4 to 3.74.0",
			version: "4.0.0",
			want:    "3.74.0",
		},
		{
			name:    "major version 4 minor 1",
			version: "4.1.0",
			want:    "4.0.0",
		},
		{
			name:    "trunk builds",
			version: "1.0.0",
			want:    "0.0.0",
		},
		{
			name:    "with nightly suffix",
			version: "3.74.x-nightly-20230224",
			want:    "3.73.0",
		},
		{
			name:    "unknown major version",
			version: "99.0.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetPreviousYStream(tt.version)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCalculateReplacedVersion(t *testing.T) {
	tests := []struct {
		name       string
		current    string
		first      string
		previous   string
		skips      []string
		unreleased string
		want       string
		wantNil    bool
	}{
		{
			name:     "downstream trunk builds get no replace",
			current:  "1.0.0",
			first:    "4.0.0",
			previous: "0.0.0",
			wantNil:  true,
		},
		{
			name:     "first release gets no replace",
			current:  "4.0.0",
			first:    "4.0.0",
			previous: "3.74.0",
			wantNil:  true,
		},
		{
			name:     "patch follows normal release",
			current:  "4.0.1",
			first:    "4.0.0",
			previous: "3.74.0",
			want:     "4.0.0",
		},
		{
			name:     "Y-Stream release replaces previous Y-Stream",
			current:  "4.2.0",
			first:    "4.0.0",
			previous: "4.1.0",
			want:     "4.1.0",
		},
		{
			name:     "normal patch replaces previous patch",
			current:  "4.1.3",
			first:    "4.0.0",
			previous: "4.0.0",
			want:     "4.1.2",
		},
		{
			name:     "first patch replaces its Y-Stream",
			current:  "4.1.1",
			first:    "4.0.0",
			previous: "4.0.0",
			want:     "4.1.0",
		},
		{
			name:     "skipped immediate preceding patch still used",
			current:  "4.1.1",
			first:    "4.0.0",
			previous: "4.0.0",
			skips:    []string{"4.1.0"},
			want:     "4.1.0",
		},
		{
			name:     "skipped immediate preceding minor patch still used",
			current:  "4.1.3",
			first:    "4.0.0",
			previous: "4.0.0",
			skips:    []string{"4.1.2"},
			want:     "4.1.2",
		},
		{
			name:     "skipped previous Y-Stream targets next patch",
			current:  "4.2.0",
			first:    "4.0.0",
			previous: "4.1.0",
			skips:    []string{"4.1.0"},
			want:     "4.1.1",
		},
		{
			name:     "multiple skips iterate to find non-skipped",
			current:  "4.3.0",
			first:    "4.0.0",
			previous: "4.2.0",
			skips:    []string{"4.1.0", "4.2.0", "4.2.1", "4.2.2", "4.4.0"},
			want:     "4.2.3",
		},
		{
			name:       "unreleased version fallback",
			current:    "4.2.0",
			first:      "4.0.0",
			previous:   "4.1.0",
			unreleased: "4.1.0",
			want:       "4.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skipVersions := make([]XyzVersion, 0)
			for _, s := range tt.skips {
				v, err := ParseXyzVersion(s)
				require.NoError(t, err)
				skipVersions = append(skipVersions, v)
			}

			got, err := CalculateReplacedVersion(tt.current, tt.first, tt.previous, skipVersions, tt.unreleased)
			require.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tt.want, got.String())
			}
		})
	}
}
