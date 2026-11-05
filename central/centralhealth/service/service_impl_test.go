package service

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	grpcerrors "github.com/stackrox/rox/pkg/grpc/errors"
	"github.com/stackrox/rox/pkg/version"
	versiontest "github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestGetUpgradeStatus(t *testing.T) {
	old := version.GetMainVersion()
	t.Cleanup(func() { versiontest.SetMainVersion(t, old) })
	for name, tc := range map[string]struct {
		target  string
		wantErr bool
	}{
		"supported":        {target: "5.1.0"},
		"missing version":  {wantErr: true},
		"invalid version":  {target: "garbage", wantErr: true},
		"missing metadata": {target: "5.5.0", wantErr: true},
	} {
		t.Run(name, func(t *testing.T) {
			versiontest.SetMainVersion(t, tc.target)
			response, err := New().GetUpgradeStatus(context.Background(), &v1.Empty{})
			if tc.wantErr {
				require.ErrorContains(t, err, "determining minimum supported Central version")
				assert.Equal(t, codes.Internal, grpcerrors.ErrToGrpcStatus(err).Code())
				assert.Nil(t, response)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.target, response.GetUpgradeStatus().GetVersion())
			assert.Equal(t, "4.10", response.GetUpgradeStatus().GetForceRollbackTo())
			assert.True(t, response.GetUpgradeStatus().GetCanRollbackAfterUpgrade())
		})
	}
}
