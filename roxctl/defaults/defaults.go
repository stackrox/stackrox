package defaults

import (
	"regexp"

	"github.com/stackrox/rox/pkg/version"
)

var (
	// ClairifyImage is the Docker image name for the "clairify" image. Image
	// repo changes depending on the main tag.
	// Example:
	// When main tag is 2.3.14.0-44-gc8a679af2b → docker.io/stackrox/clairify:0.5.2
	// When main tag is 2.3.14.1                → stackrox.io/clairify:0.5.2
	ClairifyImage = defaultClairifyImage()

	// MainImage is the Docker image name for the "main" image. Image repo
	// changes depending on the main tag.
	// Example:
	// When main tag is 2.3.14.0-44-gc8a679af2b → docker.io/stackrox/main:2.3.14.0-44-gc8a679af2b
	// When main tag is 2.3.14.1                → stackrox.io/main:2.3.14.1
	MainImage = defaultMainImage()

	// reShapshotSuffix is a compiled regex that can match git tag strings that
	// look like snapshots.
	reShapshotSuffix = regexp.MustCompilePOSIX(`-[0-9]+-g[0-9a-f]+$`)
)

func defaultClairifyImage() string {
	var (
		clairifyTag = version.GetClairifyVersion()
		mainTag     = version.GetMainVersion()
	)
	if isSnapshot(mainTag) {
		return "docker.io/stackrox/clairify:" + clairifyTag
	}
	return "stackrox.io/clairify:" + clairifyTag
}

func defaultMainImage() string {
	var mainTag = version.GetMainVersion()
	if isSnapshot(mainTag) {
		return "docker.io/stackrox/main:" + mainTag
	}
	return "stackrox.io/main:" + mainTag
}

// isSnapshot returns true if the given tag looks like a snapshot (non-release)
// git tag.
// Example:
// When tag is 2.3.14.0-44-gc8a679af2b → true (is snapshot)
// When tag is 2.3.14.1                → false (is release)
func isSnapshot(tag string) bool {
	return reShapshotSuffix.MatchString(tag)
}
