package defaults

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/pkg/version"
)

var (
	// reSnapshotSuffix is a compiled regex that can match git tag strings that
	// look like snapshots.
	reSnapshotSuffix = regexp.MustCompilePOSIX(`-[0-9]+-g[0-9a-f]+$`)
)

// ScannerImage is the Docker image name for the scanner image. Image
// repo changes depending on the main tag.
// Example:
// When main tag is 2.3.14.0-44-gc8a679af2b → docker.io/stackrox/scanner:0.5.2
// When main tag is 2.3.14.1                → stackrox.io/scanner:0.5.2
func ScannerImage() string {
	return fmt.Sprintf("%s/scanner:%s", getRegistry(), version.GetScannerVersion())
}

// MainImage is the Docker image name for the "main" image. Image repo
// changes depending on the main tag.
// Example:
// When main tag is 2.3.14.0-44-gc8a679af2b → docker.io/stackrox/main:2.3.14.0-44-gc8a679af2b
// When main tag is 2.3.14.1                → stackrox.io/main:2.3.14.1
func MainImage() string {
	return fmt.Sprintf("%s:%s", MainImageRepo(), version.GetMainVersion())
}

// MainImageRepo is the Docker image repo for the "main" image. It
// changes depending on the main tag.
// Example:
// When main tag is 2.3.14.0-44-gc8a679af2b → docker.io/stackrox/main
// When main tag is 2.3.14.1                → stackrox.io/main
func MainImageRepo() string {
	return getRegistry() + "/main"
}

func getRegistry() string {
	if isSnapshot(version.GetMainVersion()) {
		return "docker.io/stackrox"
	}
	return "stackrox.io"
}

// isSnapshot returns true if the given tag looks like a snapshot (non-release)
// git tag.
// Example:
// When tag is 2.3.14.0-44-gc8a679af2b → true (is snapshot)
// When tag is 2.3.14.1                → false (is release)
func isSnapshot(tag string) bool {
	return reSnapshotSuffix.MatchString(tag)
}
