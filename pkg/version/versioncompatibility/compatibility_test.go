package versioncompatibility

import (
	"testing"

	"github.com/stackrox/rox/pkg/version/productstreams"
	"github.com/stretchr/testify/assert"
)

const testBumpsYAML = `bumps:
  - from: "3.74"
    to: "4.0"
  - from: "4.11"
    to: "5.0"
`

func overrideTestBumps(t *testing.T) {
	productstreams.OverrideBumpsForTesting(t, []byte(testBumpsYAML))
}
func xy(x, y int) productstreams.XYVersion {
	return productstreams.XYVersion{X: x, Y: y}
}

func TestCompatibleVersionRange(t *testing.T) {
	overrideTestBumps(t)
	tests := map[string]struct {
		self productstreams.XYVersion
		n    int
		want []productstreams.XYVersion
	}{
		"mid-range no boundary": {
			self: xy(4, 5),
			n:    3,
			want: []productstreams.XYVersion{
				xy(4, 2), xy(4, 3), xy(4, 4),
				xy(4, 5),
				xy(4, 6), xy(4, 7), xy(4, 8),
			},
		},
		"crossing backward into major 3": {
			self: xy(4, 1),
			n:    3,
			want: []productstreams.XYVersion{
				xy(3, 73), xy(3, 74), xy(4, 0),
				xy(4, 1),
				xy(4, 2), xy(4, 3), xy(4, 4),
			},
		},
		"at major boundary 4.0": {
			self: xy(4, 0),
			n:    3,
			want: []productstreams.XYVersion{
				xy(3, 72), xy(3, 73), xy(3, 74),
				xy(4, 0),
				xy(4, 1), xy(4, 2), xy(4, 3),
			},
		},
		"crossing forward into major 5": {
			self: xy(4, 10),
			n:    3,
			want: []productstreams.XYVersion{
				xy(4, 7), xy(4, 8), xy(4, 9),
				xy(4, 10),
				xy(4, 11), xy(5, 0), xy(5, 1),
			},
		},
		"at bump point 4.11": {
			self: xy(4, 11),
			n:    3,
			want: []productstreams.XYVersion{
				xy(4, 8), xy(4, 9), xy(4, 10),
				xy(4, 11),
				xy(5, 0), xy(5, 1), xy(5, 2),
			},
		},
		"at major boundary 5.0": {
			self: xy(5, 0),
			n:    3,
			want: []productstreams.XYVersion{
				xy(4, 9), xy(4, 10), xy(4, 11),
				xy(5, 0),
				xy(5, 1), xy(5, 2), xy(5, 3),
			},
		},
		"5.1 spans both sides of boundary": {
			self: xy(5, 1),
			n:    3,
			want: []productstreams.XYVersion{
				xy(4, 10), xy(4, 11), xy(5, 0),
				xy(5, 1),
				xy(5, 2), xy(5, 3), xy(5, 4),
			},
		},
		"truncated backward near earliest known": {
			self: xy(3, 73),
			n:    3,
			want: []productstreams.XYVersion{
				xy(3, 70), xy(3, 71), xy(3, 72),
				xy(3, 73),
				xy(3, 74), xy(4, 0), xy(4, 1),
			},
		},
		"n=0 returns only self": {
			self: xy(4, 5),
			n:    0,
			want: []productstreams.XYVersion{xy(4, 5)},
		},
		"n=1": {
			self: xy(5, 0),
			n:    1,
			want: []productstreams.XYVersion{
				xy(4, 11),
				xy(5, 0),
				xy(5, 1),
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := CompatibleVersionRange(tt.self, tt.n)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompatibleVersions(t *testing.T) {
	overrideTestBumps(t)
	got := CompatibleVersions(xy(4, 11))
	want := []productstreams.XYVersion{
		xy(4, 8), xy(4, 9), xy(4, 10),
		xy(4, 11),
		xy(5, 0), xy(5, 1), xy(5, 2),
	}
	assert.Equal(t, want, got)
}

func TestClassify(t *testing.T) {
	overrideTestBumps(t)
	tests := map[string]struct {
		self   productstreams.XYVersion
		remote productstreams.XYVersion
		n      int
		want   Compatibility
	}{
		"matched": {
			self: xy(4, 11), remote: xy(4, 11), n: 3,
			want: Matched,
		},
		"compatible behind by 1": {
			self: xy(4, 11), remote: xy(4, 10), n: 3,
			want: CompatibleBehind,
		},
		"compatible behind by 3": {
			self: xy(4, 11), remote: xy(4, 8), n: 3,
			want: CompatibleBehind,
		},
		"compatible behind across boundary": {
			self: xy(5, 1), remote: xy(4, 10), n: 3,
			want: CompatibleBehind,
		},
		"incompatible behind by 4": {
			self: xy(4, 11), remote: xy(4, 7), n: 3,
			want: IncompatibleBehind,
		},
		"incompatible behind far": {
			self: xy(5, 1), remote: xy(3, 74), n: 3,
			want: IncompatibleBehind,
		},
		"compatible ahead by 1": {
			self: xy(4, 11), remote: xy(5, 0), n: 3,
			want: CompatibleAhead,
		},
		"compatible ahead by 3": {
			self: xy(4, 11), remote: xy(5, 2), n: 3,
			want: CompatibleAhead,
		},
		"compatible ahead across boundary": {
			self: xy(4, 9), remote: xy(5, 0), n: 3,
			want: CompatibleAhead,
		},
		"incompatible ahead by 4": {
			self: xy(4, 11), remote: xy(5, 3), n: 3,
			want: IncompatibleAhead,
		},
		"incompatible ahead far": {
			self: xy(4, 5), remote: xy(5, 3), n: 3,
			want: IncompatibleAhead,
		},
		"incompatible ahead beyond known bumps": {
			self: xy(4, 8), remote: xy(6, 0), n: 3,
			want: IncompatibleAhead,
		},
		"incompatible behind beyond known bumps": {
			self: xy(6, 0), remote: xy(4, 8), n: 3,
			want: IncompatibleBehind,
		},
		"phantom version 1 past bump point": {
			self: xy(4, 10), remote: xy(4, 12), n: 3,
			want: CompatibleAhead,
		},
		"phantom version far past bump point same major": {
			self: xy(4, 10), remote: xy(4, 40), n: 3,
			want: IncompatibleAhead,
		},
		"phantom version cross major self ahead": {
			self: xy(5, 0), remote: xy(4, 40), n: 3,
			want: IncompatibleBehind,
		},
		"phantom version in target major within skew": {
			self: xy(4, 11), remote: xy(5, 2), n: 3,
			want: CompatibleAhead,
		},
		"custom n=1 compatible": {
			self: xy(4, 5), remote: xy(4, 6), n: 1,
			want: CompatibleAhead,
		},
		"custom n=1 incompatible": {
			self: xy(4, 5), remote: xy(4, 7), n: 1,
			want: IncompatibleAhead,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := Classify(tt.self, tt.remote, tt.n)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClassifyCancelledBump(t *testing.T) {
	// Bump 5.6→6.0 is scheduled but gets cancelled. Versions 5.7, 5.8, etc.
	// are released instead.
	productstreams.OverrideBumpsForTesting(t, []byte(`bumps:
  - from: "4.11"
    to: "5.0"
  - from: "5.6"
    to: "6.0"
`))

	tests := map[string]struct {
		self   productstreams.XYVersion
		remote productstreams.XYVersion
		n      int
		want   Compatibility
	}{
		"cancelled bump remote within skew": {
			self: xy(5, 5), remote: xy(5, 8), n: 3,
			want: CompatibleAhead,
		},
		"cancelled bump remote at skew boundary": {
			self: xy(5, 5), remote: xy(5, 7), n: 3,
			want: CompatibleAhead,
		},
		"cancelled bump remote beyond skew": {
			self: xy(5, 5), remote: xy(5, 9), n: 3,
			want: IncompatibleAhead,
		},
		"cancelled bump remote behind within skew": {
			self: xy(5, 8), remote: xy(5, 5), n: 3,
			want: CompatibleBehind,
		},
		"cancelled bump remote behind beyond skew": {
			self: xy(5, 9), remote: xy(5, 5), n: 3,
			want: IncompatibleBehind,
		},
		"cancelled bump range still includes bump versions": {
			self: xy(5, 5), remote: xy(6, 0), n: 3,
			want: CompatibleAhead,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := Classify(tt.self, tt.remote, tt.n)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClassifyThreeConsecutiveBumps(t *testing.T) {
	productstreams.OverrideBumpsForTesting(t, []byte(`bumps:
  - from: "1.3"
    to: "2.0"
  - from: "2.5"
    to: "3.0"
  - from: "3.2"
    to: "4.0"
`))

	tests := map[string]struct {
		self   productstreams.XYVersion
		remote productstreams.XYVersion
		n      int
		want   Compatibility
	}{
		"chain across two bumps compatible": {
			self: xy(2, 4), remote: xy(3, 1), n: 3,
			want: CompatibleAhead,
		},
		"chain across two bumps incompatible": {
			self: xy(2, 3), remote: xy(3, 1), n: 3,
			want: IncompatibleAhead,
		},
		"chain across three bumps always incompatible": {
			self: xy(1, 3), remote: xy(4, 0), n: 3,
			want: IncompatibleAhead,
		},
		"at bump boundary matched": {
			self: xy(2, 5), remote: xy(2, 5), n: 3,
			want: Matched,
		},
		"short range near bump": {
			self: xy(3, 2), remote: xy(4, 0), n: 1,
			want: CompatibleAhead,
		},
		"short range near bump incompatible": {
			self: xy(3, 2), remote: xy(4, 1), n: 1,
			want: IncompatibleAhead,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := Classify(tt.self, tt.remote, tt.n)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClassifyNoBumps(t *testing.T) {
	productstreams.OverrideBumpsForTesting(t, []byte(`bumps: []`))

	tests := map[string]struct {
		self   productstreams.XYVersion
		remote productstreams.XYVersion
		n      int
		want   Compatibility
	}{
		"same major compatible": {
			self: xy(4, 5), remote: xy(4, 8), n: 3,
			want: CompatibleAhead,
		},
		"same major incompatible": {
			self: xy(4, 5), remote: xy(4, 9), n: 3,
			want: IncompatibleAhead,
		},
		"different major always incompatible": {
			self: xy(4, 5), remote: xy(5, 0), n: 3,
			want: IncompatibleAhead,
		},
		"matched": {
			self: xy(4, 5), remote: xy(4, 5), n: 3,
			want: Matched,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := Classify(tt.self, tt.remote, tt.n)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClassifyVersion(t *testing.T) {
	overrideTestBumps(t)
	assert.Equal(t, Matched, ClassifyVersion(xy(4, 11), xy(4, 11)))
	assert.Equal(t, CompatibleBehind, ClassifyVersion(xy(4, 11), xy(4, 9)))
	assert.Equal(t, CompatibleAhead, ClassifyVersion(xy(4, 11), xy(5, 2)))
	assert.Equal(t, IncompatibleBehind, ClassifyVersion(xy(4, 11), xy(4, 7)))
	assert.Equal(t, IncompatibleAhead, ClassifyVersion(xy(4, 11), xy(5, 3)))
}

func TestCompatibilityString(t *testing.T) {
	tests := map[string]struct {
		c    Compatibility
		want string
	}{
		"unknown":             {Unknown, "UNKNOWN"},
		"matched":             {Matched, "MATCHED"},
		"compatible behind":   {CompatibleBehind, "COMPATIBLE_BEHIND"},
		"compatible ahead":    {CompatibleAhead, "COMPATIBLE_AHEAD"},
		"incompatible behind": {IncompatibleBehind, "INCOMPATIBLE_BEHIND"},
		"incompatible ahead":  {IncompatibleAhead, "INCOMPATIBLE_AHEAD"},
		"invalid value":       {Compatibility(99), "UNKNOWN"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.c.String())
		})
	}
}
