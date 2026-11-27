package suppress_test

import (
	"net/http"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	clusterservice "github.com/stackrox/rox/central/cve/cluster/service"
	nodeservice "github.com/stackrox/rox/central/cve/node/service"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	grpcerrors "github.com/stackrox/rox/pkg/grpc/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestInvalidSuppressionDuration(t *testing.T) {
	t.Setenv(features.VulnMgmtLegacySnooze.EnvVar(), "true")
	decoder := runtime.JSONPb{UnmarshalOptions: protojson.UnmarshalOptions{DiscardUnknown: true}}

	// The gateway ignores the unknown "duraton" field, leaving duration unset.
	const payload = `{"cves":["CVE-2026-1234"],"duraton":"86400s"}`
	var req v1.SuppressCVERequest
	require.NoError(t, decoder.Unmarshal([]byte(payload), &req))
	require.Equal(t, []string{"CVE-2026-1234"}, req.GetCves())
	require.Nil(t, req.GetDuration())

	for name, request := range map[string]*v1.SuppressCVERequest{
		"misspelled duration": &req,
		"invalid nanoseconds": {Cves: req.GetCves(), Duration: &durationpb.Duration{Nanos: 1_000_000_000}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Run("node", func(t *testing.T) {
				_, err := nodeservice.New(nil).SuppressCVEs(t.Context(), request)
				require.ErrorIs(t, err, errox.InvalidArgs)
				require.Equal(t, http.StatusBadRequest, grpcerrors.ErrToHTTPStatus(err))
			})
			t.Run("cluster", func(t *testing.T) {
				_, err := clusterservice.New(nil).SuppressCVEs(t.Context(), request)
				require.ErrorIs(t, err, errox.InvalidArgs)
				require.Equal(t, http.StatusBadRequest, grpcerrors.ErrToHTTPStatus(err))
			})
		})
	}
}
