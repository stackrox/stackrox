package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testBumps = []MajorBump{
	{From: XY{3, 74}, To: XY{4, 0}},
	{From: XY{4, 11}, To: XY{5, 0}},
}

func TestParseXY(t *testing.T) {
	tests := map[string]struct {
		input   string
		want    XY
		wantErr bool
	}{
		"simple X.Y": {
			input: "4.11",
			want:  XY{4, 11},
		},
		"release X.Y.Z": {
			input: "4.11.2",
			want:  XY{4, 11},
		},
		"RC version": {
			input: "4.11.0-rc.1",
			want:  XY{4, 11},
		},
		"dev build": {
			input: "4.11.x-123-gabcdef1234",
			want:  XY{4, 11},
		},
		"major version 5": {
			input: "5.1.0",
			want:  XY{5, 1},
		},
		"old 3-component": {
			input: "3.74.1",
			want:  XY{3, 74},
		},
		"invalid": {
			input:   "not-a-version",
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ParseXY(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMinorDistance(t *testing.T) {
	tests := map[string]struct {
		a, b    XY
		bumps   []MajorBump
		want    int
		wantErr bool
	}{
		"same version": {
			a: XY{4, 5}, b: XY{4, 5},
			bumps: testBumps,
			want:  0,
		},
		"same major, 1 apart": {
			a: XY{4, 5}, b: XY{4, 6},
			bumps: testBumps,
			want:  1,
		},
		"same major, 3 apart": {
			a: XY{4, 5}, b: XY{4, 8},
			bumps: testBumps,
			want:  3,
		},
		"across 4->5 bump, adjacent": {
			a: XY{4, 11}, b: XY{5, 0},
			bumps: testBumps,
			want:  1,
		},
		"across 4->5 bump, distance 3": {
			a: XY{4, 10}, b: XY{5, 1},
			bumps: testBumps,
			want:  3,
		},
		"across 4->5 bump, distance 5": {
			a: XY{4, 9}, b: XY{5, 2},
			bumps: testBumps,
			want:  5,
		},
		"reversed order": {
			a: XY{5, 1}, b: XY{4, 10},
			bumps: testBumps,
			want:  3,
		},
		"across 3->4 bump": {
			a: XY{3, 74}, b: XY{4, 0},
			bumps: testBumps,
			want:  1,
		},
		"across 3->4 bump, distance 4": {
			a: XY{3, 73}, b: XY{4, 2},
			bumps: testBumps,
			want:  4,
		},
		"missing bump data": {
			a: XY{4, 5}, b: XY{6, 0},
			bumps:   testBumps,
			wantErr: true,
		},
		"lo version exceeds bump point": {
			a: XY{4, 12}, b: XY{5, 1},
			bumps:   testBumps,
			wantErr: true,
		},
		"hi version exceeds bump point": {
			a: XY{3, 70}, b: XY{3, 75},
			bumps:   testBumps,
			wantErr: true,
		},
		"version at bump point is valid": {
			a: XY{4, 11}, b: XY{5, 2},
			bumps: testBumps,
			want:  3,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := MinorDistance(tt.a, tt.b, tt.bumps)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCheckSkew(t *testing.T) {
	tests := map[string]struct {
		roxctl  string
		central string
		maxSkew int
		status  SkewStatus
	}{
		"identical versions": {
			roxctl: "4.11.2", central: "4.11.2",
			maxSkew: 3, status: SkewOK,
		},
		"same X.Y, different patch": {
			roxctl: "4.11.1", central: "4.11.3",
			maxSkew: 3, status: SkewOK,
		},
		"within range, 2 apart": {
			roxctl: "4.9.0", central: "4.11.0",
			maxSkew: 3, status: SkewOK,
		},
		"at boundary, 3 apart": {
			roxctl: "4.8.0", central: "4.11.0",
			maxSkew: 3, status: SkewOK,
		},
		"outside range, 4 apart": {
			roxctl: "4.7.0", central: "4.11.0",
			maxSkew: 3, status: SkewWarning,
		},
		"roxctl newer, within range": {
			roxctl: "5.1.0", central: "4.11.0",
			maxSkew: 3, status: SkewOK,
		},
		"roxctl newer, outside range": {
			roxctl: "5.3.0", central: "4.11.0",
			maxSkew: 3, status: SkewWarning,
		},
		"cross-major within range": {
			roxctl: "4.10.0", central: "5.1.0",
			maxSkew: 3, status: SkewOK,
		},
		"cross-major outside range": {
			roxctl: "4.8.0", central: "5.1.0",
			maxSkew: 3, status: SkewWarning,
		},
		"unparseable roxctl version": {
			roxctl: "garbage", central: "4.11.0",
			maxSkew: 3, status: SkewIndeterminate,
		},
		"unparseable central version": {
			roxctl: "4.11.0", central: "garbage",
			maxSkew: 3, status: SkewIndeterminate,
		},
		"dev build roxctl": {
			roxctl: "4.11.x-123-gabcdef1234", central: "4.11.0",
			maxSkew: 3, status: SkewOK,
		},
		"RC central": {
			roxctl: "4.11.0", central: "5.0.0-rc.1",
			maxSkew: 3, status: SkewOK,
		},
		"version inconsistent with bumps": {
			roxctl: "4.12.0", central: "5.1.0",
			maxSkew: 3, status: SkewIndeterminate,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := CheckSkew(tt.roxctl, tt.central, tt.maxSkew, testBumps)
			assert.Equal(t, tt.status, result.Status, "message: %s", result.Message)
		})
	}
}

func TestMergeBumps(t *testing.T) {
	a := []MajorBump{{From: XY{3, 74}, To: XY{4, 0}}}
	b := []MajorBump{
		{From: XY{3, 74}, To: XY{4, 0}},
		{From: XY{4, 11}, To: XY{5, 0}},
	}
	merged := MergeBumps(a, b)
	assert.Len(t, merged, 2)
}

func TestEmbeddedMajorBumps(t *testing.T) {
	bumps, err := EmbeddedMajorBumps()
	require.NoError(t, err)
	assert.NotEmpty(t, bumps)
	assert.Equal(t, XY{3, 74}, bumps[0].From)
	assert.Equal(t, XY{4, 0}, bumps[0].To)
}
