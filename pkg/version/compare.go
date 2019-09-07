package version

import (
	"strconv"
	"strings"

	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

func convertToIntArray(version string) (out []int) {
	splitVersion := strings.Split(version, ".")
	for _, v := range splitVersion {
		asInt, err := strconv.Atoi(v)
		if err != nil {
			log.Errorf("UNEXPECTED: got non-integer portion of version %s: %v", version, err)
			// This should never happen, but no point panic-ing here.
			// Esp since this only happens on release builds.
			// Treat all string values as -1 (which is treated as an earlier release.)
			asInt = -1
		}
		out = append(out, asInt)
	}
	return
}

func lexicographicCompareIntArrays(a, b []int) int {
	if len(a) > len(b) {
		return -lexicographicCompareIntArrays(b, a)
	}
	for idx := range a {
		if a[idx] == b[idx] {
			continue
		}
		if a[idx] < b[idx] {
			return -1
		}
		return 1
	}
	if len(b) > len(a) {
		return -1
	}
	return 0
}

// CompareReleaseVersionsOr compares the two versions if both of them are release versions.
// If at least one of the versions is not a release versions, they are incomparable UNLESS they are exactly the same,
// and if they're not exactly the same, the result specified as incomparableRes is returned.
// Otherwise, a result of < 0 is returned if versionA is lower than versionB,
// a result of > 0 is returned if versionA is higher than versionB, and 0 is returned if both versions are equal.
func CompareReleaseVersionsOr(versionA, versionB string, incomparableRes int) int {
	kindA := GetVersionKind(versionA)
	kindB := GetVersionKind(versionB)
	if versionA == versionB {
		return 0
	}
	if kindA != ReleaseKind || kindB != ReleaseKind {
		return incomparableRes
	}
	return lexicographicCompareIntArrays(convertToIntArray(versionA), convertToIntArray(versionB))
}

// CompareReleaseVersions compares the two versions, which must be release versions,
// and returns -1 if versionA is DEFINITELY a lower version than versionB,
// or 1 if versionA is DEFINITELY a greater version than versionB.
// It returns 0 unless both versions are release versions.
func CompareReleaseVersions(versionA, versionB string) int {
	return CompareReleaseVersionsOr(versionA, versionB, 0)
}
