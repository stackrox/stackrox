package service

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/version"
	versiontest "github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetUpgradeStatus(t *testing.T) {
	old := version.GetMainVersion()
	t.Cleanup(func() { versiontest.SetMainVersion(t, old) })
	for name, target := range map[string]string{
		"supported":        "5.1.0",
		"missing version":  "",
		"invalid version":  "garbage",
		"missing metadata": "5.5.0",
	} {
		t.Run(name, func(t *testing.T) {
			versiontest.SetMainVersion(t, target)
			response, err := New().GetUpgradeStatus(context.Background(), &v1.Empty{})
			if target != "5.1.0" {
				require.Error(t, err)
				assert.Equal(t, codes.Internal, status.Code(err))
				assert.Nil(t, response)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, target, response.GetUpgradeStatus().GetVersion())
			assert.Equal(t, "4.10", response.GetUpgradeStatus().GetForceRollbackTo())
			assert.True(t, response.GetUpgradeStatus().GetCanRollbackAfterUpgrade())
		})
	}
}
