package version

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompare(t *testing.T) {
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
	}

	for _, testCase := range testCases {
		c := testCase
		t.Run(fmt.Sprintf("%+v", testCase), func(t *testing.T) {
			assert.Equal(t, c.expectedResult, CompareReleaseVersions(c.versionA, c.versionB))
			assert.Equal(t, -c.expectedResult, CompareReleaseVersions(c.versionB, c.versionA))
		})
	}
}
