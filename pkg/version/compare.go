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

// CompareReleaseVersions compares the two versions, which must be release versions,
// and returns -1 if versionA is DEFINITELY a lower version than versionB,
// or 1 if versionA is DEFINITELY a greater version than versionB.
// It returns 0 unless both versions are release versions.
func CompareReleaseVersions(versionA, versionB string) int {
	kindA := GetVersionKind(versionA)
	kindB := GetVersionKind(versionB)
	if kindA != ReleaseKind || kindB != ReleaseKind {
		return 0
	}
	if versionA == versionB {
		return 0
	}
	return lexicographicCompareIntArrays(convertToIntArray(versionA), convertToIntArray(versionB))
}
