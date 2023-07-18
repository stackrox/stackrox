package version

import (
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/version/internal"
	"github.com/stretchr/testify/assert"
)

func TestParseCurrentVersion(t *testing.T) {
	_, err := parseMainVersion(GetMainVersion())
	assert.NoError(t, err)
}

func TestIsReleaseVersion(t *testing.T) {
	if buildinfo.ReleaseBuild && !buildinfo.TestBuild {
		internal.MainVersion = "1.2.3"
		assert.True(t, IsReleaseVersion())
		internal.MainVersion = "1.2.3-dirty"
		assert.False(t, IsReleaseVersion())
	} else {
		internal.MainVersion = "1.2.3"
		assert.False(t, IsReleaseVersion())
		internal.MainVersion = "1.2.3-dirty"
		assert.False(t, IsReleaseVersion())
	}
}
