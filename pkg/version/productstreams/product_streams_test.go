package productstreams

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMustParseBumpsDataPanicsOnInvalidInput(t *testing.T) {
	assert.Panics(t, func() {
		mustParseBumpsData([]byte(`[not valid yaml`))
	})
}

func TestParseBumpsData(t *testing.T) {
	tests := map[string]struct {
		input   string
		want    string
		wantErr string
	}{
		"valid single": {
			input: `
bumps:
  - from: "3.74"
    to: "4.0"`,
			want: "3.74->4.0",
		},
		"valid multiple sorted": {
			input: `
bumps:
  - from: "3.74"
    to: "4.0"
  - from: "4.11"
    to: "5.0"`,
			want: "3.74->4.0;4.11->5.0",
		},
		"valid multiple unsorted": {
			input: `
bumps:
  - from: "4.11"
    to: "5.0"
  - from: "3.74"
    to: "4.0"`,
			want: "3.74->4.0;4.11->5.0",
		},
		"duplicate to major": {
			input: `
bumps:
  - from: "3.74"
    to: "4.0"
  - from: "3.99"
    to: "4.0"`,
			wantErr: "overlapping ranges",
		},
		"duplicate from major": {
			input: `
bumps:
  - from: "3.74"
    to: "4.0"
  - from: "3.99"
    to: "5.0"`,
			wantErr: "overlapping ranges",
		},
		"overlapping ranges: containment": {
			input: `
bumps:
  - from: "4.11"
    to: "5.0"
  - from: "3.74"
    to: "6.0"`,
			wantErr: "overlapping ranges",
		},
		"overlapping ranges: partial": {
			input: `
bumps:
  - from: "3.74"
    to: "5.0"
  - from: "4.11"
    to: "6.0"`,
			wantErr: "overlapping ranges",
		},
		"from must be less than to": {
			input: `
bumps:
  - from: "5.0"
    to: "4.0"`,
			wantErr: "'from' 5.0 must be less than 'to' 4.0",
		},
		"to minor must be zero": {
			input: `
bumps:
  - from: "3.74"
    to: "4.5"`,
			wantErr: "'to' value \"4.5\" must have minor version 0",
		},
		"rejects x.y.z from": {
			input: `
bumps:
  - from: "3.74.0"
    to: "4.0"`,
			wantErr: "invalid 'from' value",
		},
		"rejects x.y.z-suffix from": {
			input: `
bumps:
  - from: "3.74.x-nightly-20230224"
    to: "4.0"`,
			wantErr: "invalid 'from' value",
		},
		"invalid from": {
			input: `
bumps:
  - from: "bad"
    to: "4.0"`,
			wantErr: "invalid 'from' value",
		},
		"invalid to": {
			input: `
bumps:
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
			result, err := parseBumpsData([]byte(tt.input))
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, formatBumps(result))
		})
	}
}

func formatBumps(bumps []parsedBump) string {
	parts := make([]string, 0, len(bumps))
	for _, b := range bumps {
		parts = append(parts, fmt.Sprintf("%s->%s", b.From, b.To))
	}
	return strings.Join(parts, ";")
}

func TestGetPreviousYStream(t *testing.T) {
	tests := map[string]struct {
		input   XYVersion
		want    XYVersion
		wantErr string
	}{
		"minor decrement": {
			input: XYVersion{X: 4, Y: 1},
			want:  XYVersion{X: 4, Y: 0},
		},
		"ordinary minor": {
			input: XYVersion{X: 5, Y: 10},
			want:  XYVersion{X: 5, Y: 9},
		},
		"large minor": {
			input: XYVersion{X: 45, Y: 67},
			want:  XYVersion{X: 45, Y: 66},
		},
		"major bump 4.0 -> 3.74": {
			input: XYVersion{X: 4, Y: 0},
			want:  XYVersion{X: 3, Y: 74},
		},
		"major bump 5.0 -> 4.11": {
			input: XYVersion{X: 5, Y: 0},
			want:  XYVersion{X: 4, Y: 11},
		},
		"unknown major": {
			input:   XYVersion{X: 99, Y: 0},
			wantErr: "don't know the previous Y-Stream for 99.0",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := GetPreviousYStream(tt.input)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
