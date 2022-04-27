package policyversion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLatestVersion(t *testing.T) {
	assert.Equal(t, CurrentVersion().String(), versions[len(versions)-1])
}

// versions in versions are sorted from old to new.
func TestVersionsOrderStrictlyAscending(t *testing.T) {
	policyVersions := make([]PolicyVersion, 0, len(versions))
	for _, v := range versions {
		policyVersions = append(policyVersions, PolicyVersion{v})
	}

	for idx := 0; idx < len(policyVersions)-1; idx++ {
		assert.True(t, Compare(policyVersions[idx], policyVersions[idx+1]) < 0, "'%+v' < '%+v'", policyVersions[idx], policyVersions[idx+1])
	}
}

func TestFromString(t *testing.T) {
	// All known versions converted without error.
	for _, v := range versions {
		_, err := FromString(v)
		assert.NoErrorf(t, err, "version: '%v'", v)
	}

	// Unknown versions cannot be converted.
	unknown := []string{"42", "unknown"}
	for _, v := range unknown {
		_, err := FromString(v)
		assert.Errorf(t, err, "version: '%v'", v)
	}
}

func TestVersionCompare(t *testing.T) {
	type TestCase struct {
		a       PolicyVersion
		b       PolicyVersion
		compare int
	}

	testCases := []TestCase{
		{PolicyVersion{versions[len(versions)-1]}, PolicyVersion{versions[len(versions)-2]}, 1},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.compare, Compare(tc.a, tc.b), "a: '%+v', b: '%+v'", tc.a, tc.b)
	}
}
