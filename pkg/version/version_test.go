package version

import (
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestVersionMatchesBuildFlavor(t *testing.T) {
	if testutils.GetBazelWorkspaceDir() == "" {
		t.Skip("not running under bazel")
	}

	versionStr := GetMainVersion()
	versionKind := GetVersionKind(versionStr)

	assert.NotEqualf(t, InvalidKind, versionKind, "version %q is of invalid kind", versionStr)

	if versionKind == RCKind || versionKind == ReleaseKind {
		assert.Truef(t, buildinfo.ReleaseBuild, "version %s of kind %v may only be built as a release build", versionStr, versionKind)
	}
	// OK to allow release build flavor for development versions - there might be a legitimate reason to test release
	// build logic without having to create a tag.
}
