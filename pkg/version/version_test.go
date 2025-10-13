package version

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/version/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCurrentVersion(t *testing.T) {
	_, err := parseMainVersion(GetMainVersion())
	assert.NoError(t, err)
}

func TestIsReleaseVersion(t *testing.T) {
	if buildinfo.ReleaseBuild && !buildinfo.TestBuild {
		internal.MainVersion = "1.2.3"
		assert.True(t, IsReleaseVersion())
		internal.MainVersion = "1.2.3-dirty"
		assert.False(t, IsReleaseVersion())
	} else {
		internal.MainVersion = "1.2.3"
		assert.False(t, IsReleaseVersion())
		internal.MainVersion = "1.2.3-dirty"
		assert.False(t, IsReleaseVersion())
	}
}

func TestIsPriorToScannerV4(t *testing.T) {
	// For readability
	isV2 := true
	wantErr := true
	tcs := []struct {
		in      string
		want    bool
		wantErr bool
	}{
		// Invalid versions
		{"", !isV2, wantErr},
		{"x.3.1", !isV2, wantErr},
		{"4.x.1", !isV2, wantErr},
		{"v4.3.1", !isV2, wantErr},

		// Scanner V2 versions
		{"3.74.0", isV2, !wantErr},
		{"4.0.0", isV2, !wantErr},
		{"4.2.x", isV2, !wantErr},
		{"4.3", isV2, !wantErr},
		{"4.3.99", isV2, !wantErr},
		{"4.3.1-1050-g8ece190c63", isV2, !wantErr},

		// Scanner V4 versions
		{"4.3.x", !isV2, !wantErr},
		{"4.3.x-1050-g8ece190c63-prerelease-ppc64le", !isV2, !wantErr},
		{"4.4", !isV2, !wantErr},
		{"4.4.0", !isV2, !wantErr},
		{"4.4.x", !isV2, !wantErr},
		{"4.4.99", !isV2, !wantErr},
		{"4.5.99-1050-blah", !isV2, !wantErr},
		{"5.0", !isV2, !wantErr},
		{"5.0.0", !isV2, !wantErr},
		{"5.0.x", !isV2, !wantErr},
		{"5.99.0", !isV2, !wantErr},
		{"5.0.99", !isV2, !wantErr},
		{"5.99.99", !isV2, !wantErr},
		{"99.99.99", !isV2, !wantErr},
	}

	for _, tc := range tcs {
		t.Run(fmt.Sprint(tc.in), func(t *testing.T) {
			got, err := IsPriorToScannerV4(tc.in)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}

}

func TestVersionVariants(t *testing.T) {
	errorTCs := []struct {
		in string
	}{
		{""},
		{"garbage"},
		{"4"},
		{"x.y"},
		{"4.y"},
	}

	for _, tc := range errorTCs {
		t.Run(fmt.Sprint(tc.in), func(t *testing.T) {
			_, err := Variants(tc.in)
			require.Error(t, err)
		})
	}

	tcs := []struct {
		in   string
		want []string
	}{
		{"4.3.2.1", []string{"4.3.2.1", "4.3.2", "4.3"}},
		{"3.74.9", []string{"3.74.9", "3.74"}},
		{"4.3.x", []string{"4.3.x", "4.3"}},
		{"4.4", []string{"4.4"}},
		{"4.3.x-1050-g8ece190c63-prerelease-ppc64le", []string{
			"4.3.x-1050-g8ece190c63-prerelease-ppc64le",
			"4.3.x-1050-g8ece190c63-prerelease",
			"4.3.x-1050-g8ece190c63",
			"4.3.x-1050",
			"4.3.x",
			"4.3",
		}},
	}

	for _, tc := range tcs {
		t.Run(fmt.Sprint(tc.in), func(t *testing.T) {
			got, err := Variants(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
