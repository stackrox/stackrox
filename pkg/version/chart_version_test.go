package version

import (
	"fmt"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/stretchr/testify/assert"
)

func TestChartVersionGeneration(t *testing.T) {
	testCases := []struct {
		mainVersion   string
		chartVersion  string
		expectedError string
	}{
		{
			mainVersion:  "3.0.49.x-1-ga0897a21ee-dirty",
			chartVersion: "300.49.0-1-ga0897a21ee-dirty",
		},
		{
			mainVersion:  "3.0.49.0-1-ga0897a21ee",
			chartVersion: "300.49.0-1-ga0897a21ee",
		},
		{
			mainVersion:  "3.0.49.1-22-ga0897a21ee",
			chartVersion: "300.49.1-22-ga0897a21ee",
		},
		{
			mainVersion:   "99.0.101.42-212-ga0897a21ee",
			expectedError: "unexpected main version",
		},
		{
			mainVersion:  "3.0.48.0-rc.1",
			chartVersion: "300.48.0-rc.1",
		},
		{
			mainVersion:  "3.0.48.5-nightly-20200910",
			chartVersion: "300.48.5-nightly-20200910",
		},
		{
			mainVersion:  "3.0.48.5",
			chartVersion: "300.48.5",
		},
		{
			mainVersion:  "3.62.x-1-ga0897a21ee-dirty",
			chartVersion: "300.62.0-1-ga0897a21ee-dirty",
		},
		{
			mainVersion:  "3.62.0-1-ga0897a21ee",
			chartVersion: "300.62.0-1-ga0897a21ee",
		},
		{
			mainVersion:  "3.62.1-22-ga0897a21ee",
			chartVersion: "300.62.1-22-ga0897a21ee",
		},
		{
			mainVersion:  "99.101.42-212-ga0897a21ee",
			chartVersion: "9900.101.42-212-ga0897a21ee",
		},
		{
			mainVersion:  "3.62.0-rc.1",
			chartVersion: "300.62.0-rc.1",
		},
		{
			mainVersion:  "3.62.5-nightly-20200910",
			chartVersion: "300.62.5-nightly-20200910",
		},
		{
			mainVersion:  "3.62.5",
			chartVersion: "300.62.5",
		},
		{
			mainVersion:   "3.62",
			expectedError: "failed to parse main version",
		},
		{
			mainVersion:  "4.0.0-rc.2",
			chartVersion: "400.0.0-rc.2",
		},
		{
			mainVersion:  "4.1.3",
			chartVersion: "400.1.3",
		},
		{
			mainVersion:  "4.0.x-79-g9abecf2368-dirty",
			chartVersion: "400.0.0-79-g9abecf2368-dirty",
		},
		{
			// Downstream trunk builds will identify themselves as such.
			// This case checks there's no error.
			mainVersion:  "1.0.0",
			chartVersion: "100.0.0",
		},
	}

	for _, testCase := range testCases {
		description := fmt.Sprintf("Checking if DeriveChartVersion(%s) = %s", testCase.mainVersion, testCase.chartVersion)
		t.Run(description, func(t *testing.T) {
			generatedChartVersion, err := deriveChartVersion(testCase.mainVersion)
			if testCase.expectedError != "" {
				assert.ErrorContains(t, err, testCase.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.chartVersion, generatedChartVersion)
				_, err = semver.NewVersion(generatedChartVersion)
				assert.NoError(t, err)
			}
		})
	}
}
