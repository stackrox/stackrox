package productstreams

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testBumpsYAML = `bumps:
  - from: "3.74"
    to: "4.0"
  - from: "4.11"
    to: "5.0"
`

func overrideTestBumps(t *testing.T) {
	OverrideBumpsForTesting(t, []byte(testBumpsYAML))
}

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

func TestGetNextYStream(t *testing.T) {
	overrideTestBumps(t)

	tests := map[string]struct {
		input XYVersion
		want  XYVersion
	}{
		"normal increment": {
			input: XYVersion{X: 4, Y: 5},
			want:  XYVersion{X: 4, Y: 6},
		},
		"bump crossing 3.74 -> 4.0": {
			input: XYVersion{X: 3, Y: 74},
			want:  XYVersion{X: 4, Y: 0},
		},
		"bump crossing 4.11 -> 5.0": {
			input: XYVersion{X: 4, Y: 11},
			want:  XYVersion{X: 5, Y: 0},
		},
		"no bump at boundary": {
			input: XYVersion{X: 5, Y: 3},
			want:  XYVersion{X: 5, Y: 4},
		},
		"version before bump point": {
			input: XYVersion{X: 4, Y: 10},
			want:  XYVersion{X: 4, Y: 11},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := GetNextYStream(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseXYFromVersionString(t *testing.T) {
	tests := map[string]struct {
		input   string
		want    XYVersion
		wantErr string
	}{
		"simple X.Y": {
			input: "4.11",
			want:  XYVersion{X: 4, Y: 11},
		},
		"X.Y.Z release": {
			input: "4.11.2",
			want:  XYVersion{X: 4, Y: 11},
		},
		"rc version": {
			input: "5.0.0-rc.1",
			want:  XYVersion{X: 5, Y: 0},
		},
		"nightly version": {
			input: "4.11.x-nightly-20210405",
			want:  XYVersion{X: 4, Y: 11},
		},
		"dev version with git hash": {
			input: "4.11.x-123-gabcdef1234",
			want:  XYVersion{X: 4, Y: 11},
		},
		"legacy 4-component format": {
			input: "3.0.61.1",
			want:  XYVersion{X: 3, Y: 61},
		},
		"legacy 4-component with x patch": {
			input: "3.0.49.x-1-ga0897a21ee",
			want:  XYVersion{X: 3, Y: 49},
		},
		"single component": {
			input:   "4",
			wantErr: "expected at least major.minor format",
		},
		"empty string": {
			input:   "",
			wantErr: "expected at least major.minor format",
		},
		"non-numeric major": {
			input:   "abc.11",
			wantErr: "invalid major",
		},
		"non-numeric minor": {
			input:   "4.abc",
			wantErr: "invalid minor",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ParseXYFromVersionString(tt.input)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDistance(t *testing.T) {
	overrideTestBumps(t)

	tests := map[string]struct {
		a    XYVersion
		b    XYVersion
		want int
	}{
		"same version": {
			a: XYVersion{X: 4, Y: 5}, b: XYVersion{X: 4, Y: 5}, want: 0,
		},
		"within same major": {
			a: XYVersion{X: 4, Y: 5}, b: XYVersion{X: 4, Y: 8}, want: 3,
		},
		"reversed order": {
			a: XYVersion{X: 4, Y: 8}, b: XYVersion{X: 4, Y: 5}, want: 3,
		},
		"across one bump": {
			a: XYVersion{X: 4, Y: 11}, b: XYVersion{X: 5, Y: 0}, want: 1,
		},
		"across bump with distance": {
			a: XYVersion{X: 4, Y: 10}, b: XYVersion{X: 5, Y: 1}, want: 3,
		},
		"across two bumps": {
			a: XYVersion{X: 3, Y: 73}, b: XYVersion{X: 5, Y: 1}, want: 15,
		},
		"adjacent": {
			a: XYVersion{X: 4, Y: 5}, b: XYVersion{X: 4, Y: 6}, want: 1,
		},
		"missing bump data": {
			a: XYVersion{X: 4, Y: 5}, b: XYVersion{X: 6, Y: 0}, want: -1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.a.Distance(tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNaiveDistance(t *testing.T) {
	overrideTestBumps(t)

	tests := map[string]struct {
		a    XYVersion
		b    XYVersion
		want int
	}{
		"same version": {
			a: XYVersion{X: 4, Y: 5}, b: XYVersion{X: 4, Y: 5}, want: 0,
		},
		"same major": {
			a: XYVersion{X: 4, Y: 5}, b: XYVersion{X: 4, Y: 8}, want: 3,
		},
		"same major reversed": {
			a: XYVersion{X: 4, Y: 8}, b: XYVersion{X: 4, Y: 5}, want: 3,
		},
		"cross major via bump": {
			a: XYVersion{X: 4, Y: 10}, b: XYVersion{X: 5, Y: 1}, want: 3,
		},
		"phantom version past bump in target major": {
			a: XYVersion{X: 4, Y: 11}, b: XYVersion{X: 5, Y: 2}, want: 3,
		},
		"phantom version far past bump": {
			a: XYVersion{X: 4, Y: 10}, b: XYVersion{X: 5, Y: 40}, want: 42,
		},
		"same major ignores bump": {
			a: XYVersion{X: 4, Y: 10}, b: XYVersion{X: 4, Y: 40}, want: 30,
		},
		"missing bump data": {
			a: XYVersion{X: 4, Y: 5}, b: XYVersion{X: 6, Y: 0}, want: -1,
		},
		"phantom past bump point cross major": {
			a: XYVersion{X: 4, Y: 12}, b: XYVersion{X: 5, Y: 0}, want: -1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.a.NaiveDistance(tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetPreviousYStream(t *testing.T) {
	overrideTestBumps(t)

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
