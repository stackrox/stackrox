package version

import (
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/stackrox/rox/pkg/logging"
)

var (
	log              = logging.LoggerForModule()
	hashTagRegex     = regexp.MustCompile(hashTagRegexStr)
	compareDevBuilds = strings.ToLower(os.Getenv("ROX_DONT_COMPARE_DEV_BUILDS")) != "true"
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
	if versionA == versionB {
		return 0
	}
	kindA := GetVersionKind(versionA)
	kindB := GetVersionKind(versionB)
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

// CompareVersions compares the two versions and returns -1 if versionA is a lower version than
// versionB or 1 if version A is a greater version than versionB.
// It returns 0 if they are not comparable.
func CompareVersions(versionA, versionB string) int {
	return CompareVersionsOr(versionA, versionB, 0)
}

// CompareVersionsOr compares the two versions and returns -1 if versionA is a lower version than
// versionB or 1 if version A is a greater version than versionB.
// It returns incomparableRes if they are not comparable.
func CompareVersionsOr(versionA, versionB string, incomparableRes int) int {
	if !compareDevBuilds {
		return CompareReleaseVersionsOr(versionA, versionB, incomparableRes)
	}

	if versionA == versionB {
		return 0
	}

	kindA := GetVersionKind(versionA)
	kindB := GetVersionKind(versionB)

	// Cannot compare invalid kind
	if kindA == InvalidKind || kindB == InvalidKind {
		return incomparableRes
	}

	if kindA == kindB {
		versionA = getEffectVersion(versionA)
		versionB = getEffectVersion(versionB)
		return CompareReleaseVersionsOr(versionA, versionB, incomparableRes)
	}

	// Compare first version part 3(.0)?.58.x
	// Because we always bump release and rc version before release date, dev and nightly version is greater than
	// release and rc version.
	// for example: 3.0.58.x-1 > 3.0.58.0-rc.1
	versionA = strings.Replace(versionA, "x", strconv.Itoa(math.MaxInt32), 1)
	versionB = strings.Replace(versionB, "x", strconv.Itoa(math.MaxInt32), 1)
	result := CompareReleaseVersionsOr(strings.Split(versionA, "-")[0], strings.Split(versionB, "-")[0], incomparableRes)
	if result != 0 {
		return result
	}

	// Release build is greater than RC build
	if kindA == ReleaseKind && kindB == RCKind {
		return 1
	}
	if kindA == RCKind && kindB == ReleaseKind {
		return -1
	}
	// Cannot compare nightly kind vs Dev kind with the same branching point
	return incomparableRes
}

func getEffectVersion(version string) string {
	// Remove hashTag
	version = hashTagRegex.ReplaceAllString(version, "")
	// Remove "-rc", 3.0.58.0-rc.1 -> 3.0.58.0.1
	version = strings.Replace(version, "-rc", "", 1)
	// Remove "-nightly", 3.0.58.x-nightly-20210405 -> 3.0.58.x-20210405
	version = strings.Replace(version, "-nightly", "", 1)
	// 3.0.58.x-189-dirty -> 3.0.58.2147483647-189-dirty to make dev build greater than release and rc build
	version = strings.Replace(version, "x", strconv.Itoa(math.MaxInt32), 1)
	// 3.0.58.2147483647-189-dirty -> 3.0.58.2147483647.189.dirty
	version = strings.Replace(version, "-", ".", -1)
	// 3.0.58.2147483647.189.dirty -> 3.0.58.2147483647.189.1, dirty version is greater than its base dev build
	version = strings.Replace(version, "dirty", "1", 1)
	return version
}
