package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stretchr/testify/require"
)

func TestUpdateScannerConfigurationInfo_LocalScanningDisabled(t *testing.T) {
	t.Setenv("ROX_LOCAL_IMAGE_SCANNING_ENABLED", "false")
	t.Setenv("ROX_DELEGATED_SCANNING_DISABLED", "false")
	t.Setenv(features.ScannerV4.EnvVar(), "true")
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ScannerV4Supported})
	t.Cleanup(func() {
		centralcaps.Set(nil)
	})

	UpdateScannerConfigurationInfo()

	require.Equal(t, float64(1), testutil.ToFloat64(scannerConfigurationInfo.WithLabelValues("false", "none", "false")))
	require.Equal(t, 1, testutil.CollectAndCount(scannerConfigurationInfo))

	require.Equal(t, float64(1), testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues("cluster_local", "central_scanner")))
	require.Equal(t, float64(1), testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues("non_cluster_local", "central_scanner")))
	require.Equal(t, 2, testutil.CollectAndCount(imageIndexingRouteInfo))
}

func TestUpdateScannerConfigurationInfo_LocalScanningEnabledScannerV2DelegatedDisabled(t *testing.T) {
	t.Setenv("ROX_LOCAL_IMAGE_SCANNING_ENABLED", "true")
	t.Setenv("ROX_DELEGATED_SCANNING_DISABLED", "true")
	t.Setenv(features.ScannerV4.EnvVar(), "false")
	centralcaps.Set(nil)
	t.Cleanup(func() {
		centralcaps.Set(nil)
	})

	UpdateScannerConfigurationInfo()

	require.Equal(t, float64(1), testutil.ToFloat64(scannerConfigurationInfo.WithLabelValues("true", "v2", "false")))
	require.Equal(t, 1, testutil.CollectAndCount(scannerConfigurationInfo))

	require.Equal(t, float64(1), testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues("cluster_local", "local_scanner")))
	require.Equal(t, float64(1), testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues("non_cluster_local", "central_scanner")))
	require.Equal(t, 2, testutil.CollectAndCount(imageIndexingRouteInfo))
}

func TestUpdateScannerConfigurationInfo_LocalScanningEnabledScannerV4DelegatedEnabled(t *testing.T) {
	t.Setenv("ROX_LOCAL_IMAGE_SCANNING_ENABLED", "true")
	t.Setenv("ROX_DELEGATED_SCANNING_DISABLED", "false")
	t.Setenv(features.ScannerV4.EnvVar(), "true")
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ScannerV4Supported})
	t.Cleanup(func() {
		centralcaps.Set(nil)
	})

	UpdateScannerConfigurationInfo()

	require.Equal(t, float64(1), testutil.ToFloat64(scannerConfigurationInfo.WithLabelValues("true", "v4", "true")))
	require.Equal(t, 1, testutil.CollectAndCount(scannerConfigurationInfo))

	require.Equal(t, float64(1), testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues("cluster_local", "local_scanner")))
	require.Equal(t, float64(1), testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues("non_cluster_local", "central_scanner")))
	require.Equal(t, 2, testutil.CollectAndCount(imageIndexingRouteInfo))
}

func TestUpdateScannerConfigurationInfo_LocalScanningEnabledScannerV2WhenCentralLacksCapability(t *testing.T) {
	t.Setenv("ROX_LOCAL_IMAGE_SCANNING_ENABLED", "true")
	t.Setenv("ROX_DELEGATED_SCANNING_DISABLED", "false")
	t.Setenv(features.ScannerV4.EnvVar(), "true")
	centralcaps.Set(nil)
	t.Cleanup(func() {
		centralcaps.Set(nil)
	})

	UpdateScannerConfigurationInfo()

	require.Equal(t, float64(1), testutil.ToFloat64(scannerConfigurationInfo.WithLabelValues("true", "v2", "true")))
	require.Equal(t, 1, testutil.CollectAndCount(scannerConfigurationInfo))
}
