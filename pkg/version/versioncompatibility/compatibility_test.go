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
		"phantom 4.12 snaps backward to bump point": {
			self: xy(4, 12),
			n:    3,
			want: []productstreams.XYVersion{
				xy(4, 9), xy(4, 10), xy(4, 11),
				xy(4, 12),
				xy(5, 0), xy(5, 1), xy(5, 2),
			},
		},
		"phantom 4.14 skips intermediates": {
			self: xy(4, 14),
			n:    3,
			want: []productstreams.XYVersion{
				xy(4, 9), xy(4, 10), xy(4, 11),
				xy(4, 14),
				xy(5, 0), xy(5, 1), xy(5, 2),
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
		"n=0 phantom self does not include bump point": {
			self: xy(4, 12),
			n:    0,
			want: []productstreams.XYVersion{xy(4, 12)},
		},
		"negative n returns only self": {
			self: xy(4, 5),
			n:    -1,
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
	type testCase struct {
		self   productstreams.XYVersion
		remote productstreams.XYVersion
		n      int
		want   Compatibility
	}

	runCases := func(t *testing.T, tests map[string]testCase) {
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				got := Classify(tt.self, tt.remote, tt.n)
				assert.Equal(t, tt.want, got)
			})
		}
	}

	// Standard bumps: 3.74→4.0, 4.11→5.0
	t.Run("standard bumps", func(t *testing.T) {
		overrideTestBumps(t)
		runCases(t, map[string]testCase{
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
				self: xy(4, 10), remote: xy(4, 11), n: 3,
				want: CompatibleAhead,
			},
			"compatible ahead by 3": {
				self: xy(4, 6), remote: xy(4, 9), n: 3,
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
			"custom n=1 compatible": {
				self: xy(4, 5), remote: xy(4, 6), n: 1,
				want: CompatibleAhead,
			},
			"custom n=1 incompatible": {
				self: xy(4, 5), remote: xy(4, 7), n: 1,
				want: IncompatibleAhead,
			},
			"negative n treats any non-self as incompatible": {
				self: xy(4, 5), remote: xy(4, 6), n: -1,
				want: IncompatibleAhead,
			},
		})
	})

	// Phantom versions: versions past a bump point (e.g. 4.12 when the
	// bump is 4.11→5.0) that exist because the bump was delayed/cancelled.
	t.Run("phantom versions", func(t *testing.T) {
		overrideTestBumps(t)
		runCases(t, map[string]testCase{
			"phantom remote within skew": {
				self: xy(4, 10), remote: xy(4, 12), n: 3,
				want: CompatibleAhead,
			},
			"phantom remote beyond skew same major": {
				self: xy(4, 7), remote: xy(4, 40), n: 3,
				want: IncompatibleAhead,
			},
			// 4.40 fits between 4.11 and 5.0 in self's compatible range,
			// so it is compatible.
			"phantom remote cross major fits in gap": {
				self: xy(5, 0), remote: xy(4, 40), n: 3,
				want: CompatibleBehind,
			},
			"phantom remote cross major gap not in range": {
				self: xy(5, 4), remote: xy(4, 40), n: 3,
				want: IncompatibleBehind,
			},
			// 4.12 is phantom but self's range ends at 4.11 — there is
			// no 5.0 after it to form a gap, so 4.12 is incompatible.
			"phantom remote at edge of range incompatible": {
				self: xy(4, 8), remote: xy(4, 12), n: 3,
				want: IncompatibleAhead,
			},
			"phantom self compatible behind": {
				self: xy(4, 12), remote: xy(4, 10), n: 3,
				want: CompatibleBehind,
			},
			"phantom self compatible ahead": {
				self: xy(4, 12), remote: xy(4, 14), n: 3,
				want: CompatibleAhead,
			},
			"phantom self incompatible behind": {
				self: xy(4, 12), remote: xy(4, 8), n: 3,
				want: IncompatibleBehind,
			},
			"phantom self cross major compatible": {
				self: xy(4, 12), remote: xy(5, 0), n: 3,
				want: CompatibleAhead,
			},
			// Symmetry: if 4.12 considers 5.0 compatible, then 5.0
			// must consider 4.12 compatible too.
			"phantom remote symmetric with phantom self": {
				self: xy(5, 0), remote: xy(4, 12), n: 3,
				want: CompatibleBehind,
			},
			"both phantom self and remote same major": {
				self: xy(4, 14), remote: xy(4, 16), n: 3,
				want: CompatibleAhead,
			},
			"phantom remote fits in gap from ahead self": {
				self: xy(5, 2), remote: xy(4, 12), n: 3,
				want: CompatibleBehind,
			},
		})
	})

	// Three consecutive bumps: 1.3→2.0, 2.5→3.0, 3.2→4.0.
	// Verifies that spanning all three is always incompatible within n=3.
	t.Run("three consecutive bumps", func(t *testing.T) {
		productstreams.OverrideBumpsForTesting(t, []byte(`bumps:
  - from: "1.3"
    to: "2.0"
  - from: "2.5"
    to: "3.0"
  - from: "3.2"
    to: "4.0"
`))
		runCases(t, map[string]testCase{
			"across all three bumps always incompatible": {
				self: xy(1, 3), remote: xy(4, 0), n: 3,
				want: IncompatibleAhead,
			},
		})
	})

	// No bumps at all: cross-major is always incompatible.
	t.Run("no bumps", func(t *testing.T) {
		productstreams.OverrideBumpsForTesting(t, []byte(`bumps: []`))
		runCases(t, map[string]testCase{
			"same major compatible": {
				self: xy(4, 5), remote: xy(4, 8), n: 3,
				want: CompatibleAhead,
			},
			"same major incompatible": {
				self: xy(4, 5), remote: xy(4, 9), n: 3,
				want: IncompatibleAhead,
			},
			"cross major always incompatible": {
				self: xy(4, 5), remote: xy(5, 0), n: 3,
				want: IncompatibleAhead,
			},
			"matched": {
				self: xy(4, 5), remote: xy(4, 5), n: 3,
				want: Matched,
			},
		})
	})
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
