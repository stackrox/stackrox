package version

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/version"
	versiontest "github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/require"
)

func TestSetCurrentVersionInvalidTarget(t *testing.T) {
	old := version.GetMainVersion()
	t.Cleanup(func() { versiontest.SetMainVersion(t, old) })
	for name, target := range map[string]string{
		"invalid version":  "garbage",
		"missing metadata": "5.5.0",
	} {
		t.Run(name, func(t *testing.T) {
			versiontest.SetMainVersion(t, target)
			_, expectedErr := migrations.MinimumSupportedForVersion(target)
			require.Error(t, expectedErr)
			require.EqualError(t, SetCurrentVersion(context.Background(), nil), expectedErr.Error())
		})
	}
}
