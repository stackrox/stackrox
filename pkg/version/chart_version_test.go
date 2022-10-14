package version

import (
	"fmt"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/stretchr/testify/assert"
)

func TestChartVersionGeneration(t *testing.T) {
	testCases := []struct {
		mainVersion  string
		chartVersion string
	}{
		{
			mainVersion:  "3.0.49.x-1-ga0897a21ee-dirty",
			chartVersion: "49.0.1-ga0897a21ee-dirty",
		},
		{
			mainVersion:  "3.0.49.0-1-ga0897a21ee",
			chartVersion: "49.0.1-ga0897a21ee",
		},
		{
			mainVersion:  "3.0.49.1-22-ga0897a21ee",
			chartVersion: "49.1.22-ga0897a21ee",
		},
		{
			mainVersion:  "99.0.101.42-212-ga0897a21ee",
			chartVersion: "101.42.212-ga0897a21ee",
		},
		{
			mainVersion:  "3.0.48.0-rc.1",
			chartVersion: "48.0.0-rc.1",
		},
		{
			mainVersion:  "3.0.48.5-nightly-20200910",
			chartVersion: "48.5.0-nightly-20200910",
		},
		{
			mainVersion:  "3.0.48.5",
			chartVersion: "48.5.0",
		},
		{
			mainVersion:  "3.62",
			chartVersion: "",
		},
		{
			mainVersion:  "3.62.x-1-ga0897a21ee-dirty",
			chartVersion: "62.0.1-ga0897a21ee-dirty",
		},
		{
			mainVersion:  "3.62.0-1-ga0897a21ee",
			chartVersion: "62.0.1-ga0897a21ee",
		},
		{
			mainVersion:  "3.62.1-22-ga0897a21ee",
			chartVersion: "62.1.22-ga0897a21ee",
		},
		{
			mainVersion:  "99.101.42-212-ga0897a21ee",
			chartVersion: "101.42.212-ga0897a21ee",
		},
		{
			mainVersion:  "3.62.0-rc.1",
			chartVersion: "62.0.0-rc.1",
		},
		{
			mainVersion:  "3.62.5-nightly-20200910",
			chartVersion: "62.5.0-nightly-20200910",
		},
		{
			mainVersion:  "3.62.5",
			chartVersion: "62.5.0",
		},
		{
			mainVersion:  "3.62",
			chartVersion: "",
		},
	}

	for _, testCase := range testCases {
		description := fmt.Sprintf("Checking if DeriveChartVersion(%s) = %s", testCase.mainVersion, testCase.chartVersion)
		t.Run(description, func(t *testing.T) {
			generatedChartVersion, err := doDeriveChartVersion(testCase.mainVersion)
			if testCase.chartVersion == "" {
				assert.ErrorContains(t, err, "failed to parse main version")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.chartVersion, generatedChartVersion)
				_, err = semver.NewVersion(generatedChartVersion)
				assert.NoError(t, err)
			}
		})
	}
}
