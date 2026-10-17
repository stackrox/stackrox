package operatorbundle

import (
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
)

// SelectLatestPatch chooses the update-candidate bundle from candidates: the one with the
// highest version that shares the installed version's major.minor and is strictly newer
// than the installed version (a patch upgrade).
//
// It returns (nil, "", nil) when no such candidate exists, with the string result being a
// human-readable reason. An error is returned only when the installed version cannot be
// parsed; candidates that fail to parse are skipped.
func SelectLatestPatch(installedVersion string, candidates []Bundle) (*Bundle, string, error) {
	installed, err := version.NewVersion(installedVersion)
	if err != nil {
		return nil, "", errors.Wrapf(err, "parsing installed bundle version %q", installedVersion)
	}
	installedSeg := installed.Segments()

	var best *Bundle
	var bestVersion *version.Version
	for i := range candidates {
		c := &candidates[i]
		cv, err := version.NewVersion(bundleVersionString(c))
		if err != nil {
			// Skip unparseable candidate versions rather than failing the whole selection.
			continue
		}
		if !sameMajorMinor(installedSeg, cv.Segments()) {
			continue
		}
		// Must be strictly newer than the installed version to be an upgrade.
		if !cv.GreaterThan(installed) {
			continue
		}
		if bestVersion == nil || cv.GreaterThan(bestVersion) {
			best = c
			bestVersion = cv
		}
	}

	if best == nil {
		return nil, "no newer patch release available within the installed major.minor", nil
	}
	return best, "", nil
}

// bundleVersionString returns the best available version string for a bundle, preferring
// the normalized Version and falling back to VersionOriginal.
func bundleVersionString(b *Bundle) string {
	if b.Version != "" {
		return b.Version
	}
	return b.VersionOriginal
}

// sameMajorMinor reports whether two version segment slices share the same major and minor
// components. Missing segments are treated as zero.
func sameMajorMinor(a, b []int) bool {
	return segment(a, 0) == segment(b, 0) && segment(a, 1) == segment(b, 1)
}

func segment(s []int, i int) int {
	if i < len(s) {
		return s[i]
	}
	return 0
}
