package downloaddb

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		{"3.99.99", isV2, !wantErr},
		{"4.0.0", isV2, !wantErr},
		{"4.3", isV2, !wantErr},
		{"4.3.99", isV2, !wantErr},
		{"4.3.1-1050-g8ece190c63", isV2, !wantErr},

		// Scanner V4 versions
		{"4.3.x", !isV2, !wantErr},
		{"4.3.x-1050-g8ece190c63-prerelease-ppc64le", !isV2, !wantErr},
		{"4.4", !isV2, !wantErr},
		{"4.4.0", !isV2, !wantErr},
		{"4.4.99", !isV2, !wantErr},
		{"4.5.0", !isV2, !wantErr},
		{"4.5.99", !isV2, !wantErr},
		{"4.5.99-1050-blah", !isV2, !wantErr},
	}

	for i, tc := range tcs {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			got, err := isPriorToScannerV4(tc.in)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}

}

func TestDisectVersion(t *testing.T) {
	tcs := []struct {
		in   string
		want []string
	}{
		// edge cases
		{"", []string{""}},
		{"garbage", []string{"garbage"}},
		{"4", []string{"4"}},

		// expected version
		{"4.3.2.1", []string{"4.3.2.1", "4.3.2", "4.3"}},
		{"3.74.9", []string{"3.74.9", "3.74"}},
		{"4.3.x", []string{"4.3.x", "4.3"}},
		{"4.3.x-1050-g8ece190c63-prerelease-ppc64le", []string{
			"4.3.x-1050-g8ece190c63-prerelease-ppc64le",
			"4.3.x-1050-g8ece190c63-prerelease",
			"4.3.x-1050-g8ece190c63",
			"4.3.x-1050",
			"4.3.x",
			"4.3",
		}},
		{"4.4", []string{"4.4"}},
		{"4.4.0", []string{"4.4.0", "4.4"}},
		{"4.4.99", []string{"4.4.99", "4.4"}},
		{"4.5.0", []string{"4.5.0", "4.5"}},
		{"4.5.99", []string{"4.5.99", "4.5"}},
		{"4.5.99-1050-blah", []string{
			"4.5.99-1050-blah",
			"4.5.99-1050",
			"4.5.99",
			"4.5",
		}},
	}

	for _, tc := range tcs {
		t.Run(fmt.Sprint(tc.in), func(t *testing.T) {
			got := disectVersion(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
}
