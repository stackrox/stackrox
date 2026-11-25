package suppress_test

import (
	"context"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	clusterservice "github.com/stackrox/rox/central/cve/cluster/service"
	nodeservice "github.com/stackrox/rox/central/cve/node/service"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestMisspelledSuppressionDuration(t *testing.T) {
	t.Setenv(features.VulnMgmtLegacySnooze.EnvVar(), "true")
	decoder := runtime.JSONPb{UnmarshalOptions: protojson.UnmarshalOptions{DiscardUnknown: true}}

	for name, suppress := range map[string]func(context.Context, *v1.SuppressCVERequest) (*v1.Empty, error){
		"node":    nodeservice.New(nil).SuppressCVEs,
		"cluster": clusterservice.New(nil).SuppressCVEs,
	} {
		t.Run(name, func(t *testing.T) {
			var req v1.SuppressCVERequest
			require.NoError(t, decoder.Unmarshal([]byte(`{"cves":["CVE-2026-1234"],"duraton":"86400s"}`), &req))
			_, err := suppress(t.Context(), &req)
			require.ErrorIs(t, err, errox.InvalidArgs)
		})
	}
}
