//go:build !release || test
// +build !release test

package centralsensor

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/testutils/utils"
)

// AppendSpecificVersionInfoToContext appends a version info to the context that embeds the specific version passed.
// USE ONLY IN TESTING.
// Enforced by build tag -- code that calls this will NOT compile on release builds.
func AppendSpecificVersionInfoToContext(ctx context.Context, version string, t *testing.T) (context.Context, error) {
	utils.MustBeInTest(t)
	return appendSensorVersionInfoToContext(ctx, version)
}
