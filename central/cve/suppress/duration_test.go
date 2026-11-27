package suppress_test

import (
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	clusterservice "github.com/stackrox/rox/central/cve/cluster/service"
	nodeservice "github.com/stackrox/rox/central/cve/node/service"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestMisspelledSuppressionDuration(t *testing.T) {
	t.Setenv(features.VulnMgmtLegacySnooze.EnvVar(), "true")
	decoder := runtime.JSONPb{UnmarshalOptions: protojson.UnmarshalOptions{DiscardUnknown: true}}

	// The gateway ignores the unknown "duraton" field, leaving duration unset.
	const payload = `{"cves":["CVE-2026-1234"],"duraton":"86400s"}`
	var req v1.SuppressCVERequest
	require.NoError(t, decoder.Unmarshal([]byte(payload), &req))
	require.Equal(t, []string{"CVE-2026-1234"}, req.GetCves())
	require.Nil(t, req.GetDuration())

	t.Run("node", func(t *testing.T) {
		_, err := nodeservice.New(nil).SuppressCVEs(t.Context(), &req)
		require.ErrorContains(t, err, "invalid nil Duration")
	})
	t.Run("cluster", func(t *testing.T) {
		_, err := clusterservice.New(nil).SuppressCVEs(t.Context(), &req)
		require.ErrorContains(t, err, "invalid nil Duration")
	})
}
