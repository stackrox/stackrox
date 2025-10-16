package helmconfig

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	clusterConfig := []byte(`
clusterName: remote
clusterConfig:
  staticConfig:
    type: KUBERNETES_CLUSTER
    mainImage: stackrox/main
    collectorImage: stackrox/collector
    centralApiEndpoint: central.stackrox:443
    collectionMethod: CORE_BPF
    admissionController: true
    admissionControllerUpdates: false
    admissionControllerEvents: true
    tolerationsConfig:
      disabled: false
    slimCollector: false
  dynamicConfig:
    disableAuditLogs: true
    admissionControllerConfig:
      enabled: false
      timeoutSeconds: 3
      scanInline: false
      disableBypass: false
      enforceOnUpdates: false
    registryOverride:
  clusterLabels:
    my-label1: my value 1
    my-label2: my value 2
  configFingerprint: 69c6a7ea9452e9dc13aaf7d29e2b9ac4207a53d95b900b3853dce46f47df8407-1
`)

	config, err := load(clusterConfig)
	require.NoError(t, err)

	expectedClusterConfig := storage.CompleteClusterConfig_builder{
		DynamicConfig: storage.DynamicClusterConfig_builder{
			AdmissionControllerConfig: storage.AdmissionControllerConfig_builder{
				Enabled:          false,
				TimeoutSeconds:   3,
				ScanInline:       false,
				DisableBypass:    false,
				EnforceOnUpdates: false,
			}.Build(),
			RegistryOverride: "",
			DisableAuditLogs: true,
		}.Build(),
		StaticConfig: storage.StaticClusterConfig_builder{
			Type:                       storage.ClusterType_KUBERNETES_CLUSTER,
			MainImage:                  "stackrox/main",
			CentralApiEndpoint:         "central.stackrox:443",
			CollectionMethod:           storage.CollectionMethod_CORE_BPF,
			CollectorImage:             "stackrox/collector",
			AdmissionController:        true,
			AdmissionControllerUpdates: false,
			TolerationsConfig:          storage.TolerationsConfig_builder{Disabled: false}.Build(),
			AdmissionControllerEvents:  true,
		}.Build(),
		ConfigFingerprint: "69c6a7ea9452e9dc13aaf7d29e2b9ac4207a53d95b900b3853dce46f47df8407-1",
		ClusterLabels:     map[string]string{"my-label1": "my value 1", "my-label2": "my value 2"},
	}.Build()

	expectedConfig := &central.HelmManagedConfigInit{}
	expectedConfig.SetClusterConfig(expectedClusterConfig)
	expectedConfig.SetClusterName("remote")
	expectedConfig.SetClusterId("")

	assert.True(t, expectedConfig.EqualVT(config), "Converted proto and expected proto do not match")
}
