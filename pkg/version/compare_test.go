package version

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompareReleaseVersion(t *testing.T) {
	testCases := []struct {
		versionA       string
		versionB       string
		expectedResult int
	}{
		{
			"2.5.25.0-rc.1-375-g7ab3c70477",
			"2.5.25.0-rc.1-376-g7ab3c70473",
			0,
		},
		{
			"2.5.25.0-rc.1-375-g7ab3c70477",
			"2.5.27.0",
			0,
		},
		{
			"2.5.27.0",
			"2.5.27.0",
			0,
		},
		{
			"2.5.27.0",
			"2.5.28.0",
			-1,
		},
		{
			"2.5.27",
			"2.5.27.1",
			-1,
		},
		{
			"2.5.27.2",
			"2.5.27.1",
			1,
		},
		{
			"3.0.0",
			"2.5.27.1",
			1,
		},
		{
			"2.4.23.9",
			"2.4.24.0",
			-1,
		},
		// This test case is not representative of reality, but meh.
		// Just making sure the code does what it looks like it's doing.
		{
			"2.4.23.9",
			"2.5.1.0",
			-1,
		},
		{
			"3.62.0-rc.1-375-g7ab3c70477",
			"3.62.0-rc.1-376-g7ab3c70473",
			0,
		},
		{
			"3.62.0-rc.1-375-g7ab3c70477",
			"3.63.0",
			0,
		},
		{
			"3.62.0",
			"3.62.0",
			0,
		},
		{
			"3.0.61.1",
			"3.62.0",
			-1,
		},
		{
			"3.62.0",
			"3.63.0",
			-1,
		},
		{
			"3.62",
			"3.62.1",
			-1,
		},
		{
			"3.62.2",
			"3.62.1",
			1,
		},
		{
			"4.0.0",
			"3.62.1",
			1,
		},
		{
			"3.62.9",
			"3.63.1",
			-1,
		},
		{
			"3.62.1",
			"4.10.0",
			-1,
		},
	}

	for _, testCase := range testCases {
		c := testCase
		t.Run(fmt.Sprintf("%+v", testCase), func(t *testing.T) {
			assert.Equal(t, c.expectedResult, CompareReleaseVersions(c.versionA, c.versionB))
			assert.Equal(t, -c.expectedResult, CompareReleaseVersions(c.versionB, c.versionA))
		})
	}
}

func TestCompareAnyVersion(t *testing.T) {
	testCases := []struct {
		versionA       string
		versionB       string
		expectedResult int
		incomparable   bool
	}{
		// Existing test cases for release version compare.
		{
			versionA:       "2.5.25.0-rc.1-375-g7ab3c70477",
			versionB:       "2.5.25.0-rc.1-376-g7ab3c70473",
			expectedResult: -1,
		},
		{
			versionA:       "2.5.25.0-rc.1-375-g7ab3c70477",
			versionB:       "2.5.27.0",
			expectedResult: -1,
		},
		{
			versionA:       "2.5.27.0",
			versionB:       "2.5.27.0",
			expectedResult: 0,
		},
		{
			versionA:       "2.5.27.0",
			versionB:       "2.5.28.0",
			expectedResult: -1,
		},
		{
			versionA:       "2.5.27",
			versionB:       "2.5.27.1",
			expectedResult: -1,
		},
		{
			versionA:       "2.5.27.2",
			versionB:       "2.5.27.1",
			expectedResult: 1,
		},
		{
			versionA:       "3.0.0",
			versionB:       "2.5.27.1",
			expectedResult: 1,
		},
		{
			versionA:       "2.4.23.9",
			versionB:       "2.4.24.0",
			expectedResult: -1,
		},
		{
			versionA:       "2.4.23.9",
			versionB:       "2.5.1.0",
			expectedResult: -1,
		},
		{
			versionA:       "3.0.58.0",
			versionB:       "3.0.58.0",
			expectedResult: 0,
		},
		// Compare RCKind releases
		{
			versionA:       "3.0.57.0-rc.1",
			versionB:       "3.0.56.0-rc.2",
			expectedResult: 1,
		},
		{
			versionA:       "3.0.58.0-rc.2",
			versionB:       "3.0.58.0-rc.1",
			expectedResult: 1,
		},
		{
			versionA:       "3.0.58.0-rc.1",
			versionB:       "3.0.58.0-rc.1",
			expectedResult: 0,
		},
		{
			versionA:       "3.0.58.0",
			versionB:       "3.0.58.0-rc.2",
			expectedResult: 1,
		},
		// Compare with DevelopmentKind
		{
			versionA:       "3.0.58.x-14-gd023697df1",
			versionB:       "3.0.58.x-15-gd023697df2",
			expectedResult: -1,
		},
		{
			versionA:       "3.0.59.x-14-gd023697df1",
			versionB:       "3.0.58.x-15-gd023697df1-dirty",
			expectedResult: 1,
		},
		{
			versionA:       "3.0.58.x-13-gd023697df1",
			versionB:       "3.0.58.x-15-gd023997df1-dirty",
			expectedResult: -1,
		},
		{
			versionA:       "3.0.58.x-15-gd023697df1-dirty",
			versionB:       "3.0.58.x-15-gd023697df1",
			expectedResult: 1,
		},
		{
			versionA:       "3.0.58.x-1-gd023697df1",
			versionB:       "3.0.58.0-rc.2",
			expectedResult: 1,
		},
		{
			versionA:       "3.0.58.x-13-gd023697df1-dirty",
			versionB:       "3.0.58.2",
			expectedResult: 1,
		},
		{
			versionA:       "3.0.58.x-140-gc520327875-dirty",
			versionB:       "3.0.59.0",
			expectedResult: -1,
		},
		// Compare with nightly
		{
			versionA:       "3.0.58.x-nightly-20210405",
			versionB:       "3.0.58.x-nightly-20210305",
			expectedResult: 1,
		},
		{
			versionA:     "3.0.58.x-nightly-20210405",
			versionB:     "3.0.58.x-140-gc520327875-dirty",
			incomparable: true,
		},
		{
			versionA:       "3.0.57.x-nightly-20210405",
			versionB:       "3.0.58.x-140-gc520327875-dirty",
			expectedResult: -1,
		},
		{
			versionA:       "3.0.58.x-nightly-20210405",
			versionB:       "3.0.57.1",
			expectedResult: 1,
		},
		{
			versionA:       "3.0.58.x-nightly-20210405",
			versionB:       "3.0.58.1",
			expectedResult: 1,
		},
		{
			versionA:       "3.0.58.x-nightly-20210405",
			versionB:       "3.0.58.1-rc.2",
			expectedResult: 1,
		},
		{
			versionA:     "3.0.58.x-nightly-20210405",
			versionB:     "3.0.58.x-140-gc520327875-dirty",
			incomparable: true,
		},
		{
			versionA:     "3.0.58.x-some-20210405",
			versionB:     "3.0.58.x-140-other-dirty",
			incomparable: true,
		},
		// Compare with nightly - new semver-based scheme
		{
			versionA:       "3.62.x-nightly-20210405",
			versionB:       "3.62.x-nightly-20210305",
			expectedResult: 1,
		},
		{
			versionA:     "3.62.x-nightly-20210405",
			versionB:     "3.62.x-140-gc520327875-dirty",
			incomparable: true,
		},
		{
			versionA:       "3.62.x-nightly-20210405",
			versionB:       "3.63.x-140-gc520327875-dirty",
			expectedResult: -1,
		},
		{
			versionA:       "3.63.x-nightly-20210405",
			versionB:       "3.62.1",
			expectedResult: 1,
		},
		{
			versionA:       "3.62.x-nightly-20210405",
			versionB:       "3.62.1",
			expectedResult: 1,
		},
		{
			versionA:       "3.62.x-nightly-20210405",
			versionB:       "3.62.1-rc.2",
			expectedResult: 1,
		},
		{
			versionA:     "3.62.x-nightly-20210405",
			versionB:     "3.62.x-140-gc520327875-dirty",
			incomparable: true,
		},
		{
			versionA:     "3.62.x-some-20210405",
			versionB:     "3.62.x-140-other-dirty",
			incomparable: true,
		},
		// Compare with nightly - mixed old/new
		{
			versionA:       "3.0.61.x-nightly-20210405",
			versionB:       "3.62.x-nightly-20210305",
			expectedResult: -1,
		},
		{
			versionA:       "3.0.61.x-nightly-20210405",
			versionB:       "3.62.x-140-gc520327875-dirty",
			expectedResult: -1,
		},
		{
			versionA:       "3.62.x-nightly-20210405",
			versionB:       "3.0.61.1",
			expectedResult: 1,
		},
		{
			// Even though 4.6.1 > 4.6.0, dev builds are always considered greater than release builds
			// with the same x.y (i.e. major and minor) numbers.
			versionA:       "4.6.1",
			versionB:       "4.6.0-3-gabc1233456",
			expectedResult: -1,
		},
	}

	for _, testCase := range testCases {
		c := testCase
		t.Run(fmt.Sprintf("%+v", testCase), func(t *testing.T) {
			if !c.incomparable {
				assert.Equal(t, GetVersionKind(getEffectVersion(c.versionA)), ReleaseKind)
				assert.Equal(t, GetVersionKind(getEffectVersion(c.versionB)), ReleaseKind)
			}

			if c.incomparable {
				assert.Equal(t, 9, CompareVersionsOr(c.versionA, c.versionB, 9))
				assert.Equal(t, 9, CompareVersionsOr(c.versionB, c.versionA, 9))
			} else {
				assert.Equal(t, c.expectedResult, CompareVersionsOr(c.versionA, c.versionB, 9))
				assert.Equal(t, -c.expectedResult, CompareVersionsOr(c.versionB, c.versionA, 9))
			}
			assert.Equal(t, c.expectedResult, CompareVersions(c.versionA, c.versionB))
			assert.Equal(t, -c.expectedResult, CompareVersions(c.versionB, c.versionA))
		})
	}
}
