package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stretchr/testify/assert"
)

func setupCentralCaps(t *testing.T, caps []centralsensor.CentralCapability) {
	centralcaps.Set(caps)
	t.Cleanup(func() { centralcaps.Set(nil) })
	t.Cleanup(resetLastDelegatedConfig)
}

func TestUpdateScannerConfigurationInfo_LocalScanningDisabled(t *testing.T) {
	t.Setenv("ROX_LOCAL_IMAGE_SCANNING_ENABLED", "false")
	t.Setenv("ROX_DELEGATED_SCANNING_DISABLED", "false")
	t.Setenv(features.ScannerV4.EnvVar(), "true")
	setupCentralCaps(t, []centralsensor.CentralCapability{centralsensor.ScannerV4Supported})

	UpdateScannerConfigurationInfo(nil)

	assert.InDelta(t, 1, testutil.ToFloat64(scannerConfigurationInfo.WithLabelValues("false", ModeNone, "false")), 0)
	assert.Equal(t, 1, testutil.CollectAndCount(scannerConfigurationInfo))

	assert.InDelta(t, 1, testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues(ForImagesClusterLocal, IndexerCentralScanner)), 0)
	assert.InDelta(t, 1, testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues(ForImagesNonClusterLocal, IndexerCentralScanner)), 0)
	assert.Equal(t, 2, testutil.CollectAndCount(imageIndexingRouteInfo))
}

func TestUpdateScannerConfigurationInfo_LocalScanningEnabledV2_DelegatedEnvDisabled(t *testing.T) {
	t.Setenv("ROX_LOCAL_IMAGE_SCANNING_ENABLED", "true")
	t.Setenv("ROX_DELEGATED_SCANNING_DISABLED", "true")
	t.Setenv(features.ScannerV4.EnvVar(), "false")
	setupCentralCaps(t, nil)

	UpdateScannerConfigurationInfo(nil)

	assert.InDelta(t, 1, testutil.ToFloat64(scannerConfigurationInfo.WithLabelValues("true", ModeV2, "false")), 0)
	assert.Equal(t, 1, testutil.CollectAndCount(scannerConfigurationInfo))

	assert.InDelta(t, 1, testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues(ForImagesClusterLocal, IndexerLocalScanner)), 0)
	assert.InDelta(t, 1, testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues(ForImagesNonClusterLocal, IndexerCentralScanner)), 0)
	assert.Equal(t, 2, testutil.CollectAndCount(imageIndexingRouteInfo))
}

func TestUpdateScannerConfigurationInfo_LocalScanningEnabledV4_DelegatedEnvEnabled_NoCentralConfig(t *testing.T) {
	t.Setenv("ROX_LOCAL_IMAGE_SCANNING_ENABLED", "true")
	t.Setenv("ROX_DELEGATED_SCANNING_DISABLED", "false")
	t.Setenv(features.ScannerV4.EnvVar(), "true")
	setupCentralCaps(t, []centralsensor.CentralCapability{centralsensor.ScannerV4Supported})

	UpdateScannerConfigurationInfo(nil)

	assert.InDelta(t, 1, testutil.ToFloat64(scannerConfigurationInfo.WithLabelValues("true", ModeV4, "false")), 0)
	assert.Equal(t, 1, testutil.CollectAndCount(scannerConfigurationInfo))

	assert.InDelta(t, 1, testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues(ForImagesClusterLocal, IndexerLocalScanner)), 0)
	assert.InDelta(t, 1, testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues(ForImagesNonClusterLocal, IndexerCentralScanner)), 0)
	assert.Equal(t, 2, testutil.CollectAndCount(imageIndexingRouteInfo))
}

func TestUpdateScannerConfigurationInfo_DelegatedConfigNone(t *testing.T) {
	t.Setenv("ROX_LOCAL_IMAGE_SCANNING_ENABLED", "true")
	t.Setenv("ROX_DELEGATED_SCANNING_DISABLED", "false")
	t.Setenv(features.ScannerV4.EnvVar(), "true")
	setupCentralCaps(t, []centralsensor.CentralCapability{centralsensor.ScannerV4Supported})

	config := &central.DelegatedRegistryConfig{EnabledFor: central.DelegatedRegistryConfig_NONE}
	UpdateScannerConfigurationInfo(config)

	assert.InDelta(t, 1, testutil.ToFloat64(scannerConfigurationInfo.WithLabelValues("true", ModeV4, "false")), 0)
	assert.Equal(t, 1, testutil.CollectAndCount(scannerConfigurationInfo))

	assert.InDelta(t, 1, testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues(ForImagesClusterLocal, IndexerLocalScanner)), 0)
	assert.InDelta(t, 1, testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues(ForImagesNonClusterLocal, IndexerCentralScanner)), 0)
	assert.Equal(t, 2, testutil.CollectAndCount(imageIndexingRouteInfo))
}

func TestUpdateScannerConfigurationInfo_DelegatedConfigAll(t *testing.T) {
	t.Setenv("ROX_LOCAL_IMAGE_SCANNING_ENABLED", "true")
	t.Setenv("ROX_DELEGATED_SCANNING_DISABLED", "false")
	t.Setenv(features.ScannerV4.EnvVar(), "true")
	setupCentralCaps(t, []centralsensor.CentralCapability{centralsensor.ScannerV4Supported})

	config := &central.DelegatedRegistryConfig{EnabledFor: central.DelegatedRegistryConfig_ALL}
	UpdateScannerConfigurationInfo(config)

	assert.InDelta(t, 1, testutil.ToFloat64(scannerConfigurationInfo.WithLabelValues("true", ModeV4, "true")), 0)
	assert.Equal(t, 1, testutil.CollectAndCount(scannerConfigurationInfo))

	assert.InDelta(t, 1, testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues(ForImagesClusterLocal, IndexerLocalScanner)), 0)
	assert.InDelta(t, 1, testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues(ForImagesNonClusterLocal, IndexerLocalScanner)), 0)
	assert.Equal(t, 2, testutil.CollectAndCount(imageIndexingRouteInfo))
}

func TestUpdateScannerConfigurationInfo_DelegatedConfigSpecific(t *testing.T) {
	t.Setenv("ROX_LOCAL_IMAGE_SCANNING_ENABLED", "true")
	t.Setenv("ROX_DELEGATED_SCANNING_DISABLED", "false")
	t.Setenv(features.ScannerV4.EnvVar(), "true")
	setupCentralCaps(t, []centralsensor.CentralCapability{centralsensor.ScannerV4Supported})

	config := &central.DelegatedRegistryConfig{
		EnabledFor: central.DelegatedRegistryConfig_SPECIFIC,
		Registries: []*central.DelegatedRegistryConfig_DelegatedRegistry{
			{Path: "quay.io/myorg"},
		},
	}
	UpdateScannerConfigurationInfo(config)

	assert.InDelta(t, 1, testutil.ToFloat64(scannerConfigurationInfo.WithLabelValues("true", ModeV4, "true")), 0)
	assert.Equal(t, 1, testutil.CollectAndCount(scannerConfigurationInfo))

	assert.InDelta(t, 1, testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues(ForImagesClusterLocal, IndexerLocalScanner)), 0)
	assert.InDelta(t, 1, testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues(ForImagesNonClusterLocal, IndexerMixed)), 0)
	assert.Equal(t, 2, testutil.CollectAndCount(imageIndexingRouteInfo))
}

func TestUpdateScannerConfigurationInfo_V2FallbackWhenCentralLacksV4Capability(t *testing.T) {
	t.Setenv("ROX_LOCAL_IMAGE_SCANNING_ENABLED", "true")
	t.Setenv("ROX_DELEGATED_SCANNING_DISABLED", "false")
	t.Setenv(features.ScannerV4.EnvVar(), "true")
	setupCentralCaps(t, nil)

	config := &central.DelegatedRegistryConfig{EnabledFor: central.DelegatedRegistryConfig_ALL}
	UpdateScannerConfigurationInfo(config)

	assert.InDelta(t, 1, testutil.ToFloat64(scannerConfigurationInfo.WithLabelValues("true", ModeV2, "true")), 0)
	assert.Equal(t, 1, testutil.CollectAndCount(scannerConfigurationInfo))
}

func TestUpdateScannerConfigurationInfo_DelegatedConfigAllButEnvDisabled(t *testing.T) {
	t.Setenv("ROX_LOCAL_IMAGE_SCANNING_ENABLED", "true")
	t.Setenv("ROX_DELEGATED_SCANNING_DISABLED", "true")
	t.Setenv(features.ScannerV4.EnvVar(), "true")
	setupCentralCaps(t, []centralsensor.CentralCapability{centralsensor.ScannerV4Supported})

	config := &central.DelegatedRegistryConfig{EnabledFor: central.DelegatedRegistryConfig_ALL}
	UpdateScannerConfigurationInfo(config)

	assert.InDelta(t, 1, testutil.ToFloat64(scannerConfigurationInfo.WithLabelValues("true", ModeV4, "false")), 0)
	assert.Equal(t, 1, testutil.CollectAndCount(scannerConfigurationInfo))

	assert.InDelta(t, 1, testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues(ForImagesNonClusterLocal, IndexerCentralScanner)), 0)
}

func TestUpdateScannerConfigurationInfo_SequentialUpdates(t *testing.T) {
	t.Setenv("ROX_LOCAL_IMAGE_SCANNING_ENABLED", "true")
	t.Setenv("ROX_DELEGATED_SCANNING_DISABLED", "false")
	t.Setenv(features.ScannerV4.EnvVar(), "true")
	setupCentralCaps(t, []centralsensor.CentralCapability{centralsensor.ScannerV4Supported})

	UpdateScannerConfigurationInfo(nil)
	assert.InDelta(t, 1, testutil.ToFloat64(scannerConfigurationInfo.WithLabelValues("true", ModeV4, "false")), 0)
	assert.InDelta(t, 1, testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues(ForImagesNonClusterLocal, IndexerCentralScanner)), 0)

	config := &central.DelegatedRegistryConfig{EnabledFor: central.DelegatedRegistryConfig_ALL}
	UpdateScannerConfigurationInfo(config)
	assert.InDelta(t, 1, testutil.ToFloat64(scannerConfigurationInfo.WithLabelValues("true", ModeV4, "true")), 0)
	assert.InDelta(t, 1, testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues(ForImagesNonClusterLocal, IndexerLocalScanner)), 0)

	assert.Equal(t, 1, testutil.CollectAndCount(scannerConfigurationInfo))
	assert.Equal(t, 2, testutil.CollectAndCount(imageIndexingRouteInfo))
}

func TestUpdateScannerConfigurationInfo_NilOnReconnectPreservesLastConfig(t *testing.T) {
	t.Setenv("ROX_LOCAL_IMAGE_SCANNING_ENABLED", "true")
	t.Setenv("ROX_DELEGATED_SCANNING_DISABLED", "false")
	t.Setenv(features.ScannerV4.EnvVar(), "true")
	setupCentralCaps(t, []centralsensor.CentralCapability{centralsensor.ScannerV4Supported})

	config := &central.DelegatedRegistryConfig{EnabledFor: central.DelegatedRegistryConfig_ALL}
	UpdateScannerConfigurationInfo(config)
	assert.InDelta(t, 1, testutil.ToFloat64(scannerConfigurationInfo.WithLabelValues("true", ModeV4, "true")), 0)
	assert.InDelta(t, 1, testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues(ForImagesNonClusterLocal, IndexerLocalScanner)), 0)

	// Simulate reconnect: hello handshake passes nil. Metrics must retain
	// the previously known delegated config instead of resetting.
	UpdateScannerConfigurationInfo(nil)
	assert.InDelta(t, 1, testutil.ToFloat64(scannerConfigurationInfo.WithLabelValues("true", ModeV4, "true")), 0)
	assert.InDelta(t, 1, testutil.ToFloat64(imageIndexingRouteInfo.WithLabelValues(ForImagesNonClusterLocal, IndexerLocalScanner)), 0)

	assert.Equal(t, 1, testutil.CollectAndCount(scannerConfigurationInfo))
	assert.Equal(t, 2, testutil.CollectAndCount(imageIndexingRouteInfo))
}
