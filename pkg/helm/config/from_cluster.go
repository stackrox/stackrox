package config

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/renderer"
)

// imageSpecFromOverrides produces an image spec to be used in the Secured Cluster Helm chart configuration,
// given a map returned by `renderer.ComputeImageOverrides`.
func imageSpecFromOverrides(overrides map[string]string) map[string]interface{} {
	image := make(map[string]interface{})

	if val := overrides["Registry"]; val != "" {
		image["registry"] = val
	}
	if val := overrides["Name"]; val != "" {
		image["name"] = val
	}
	if val := overrides["Tag"]; val != "" && val != "latest" {
		image["tag"] = val
	}

	return image
}

// FromCluster returns the cluster's Helm chart configuration based on cluster and image flavor.
func FromCluster(cluster *storage.Cluster, flavor defaults.ImageFlavor) (map[string]interface{}, error) {
	mainImageOverrides := renderer.ComputeImageOverrides(cluster.GetMainImage(), flavor.MainRegistry, flavor.MainImageName, "")
	mainImage := imageSpecFromOverrides(mainImageOverrides)
	collectorImageOverrides := renderer.ComputeImageOverrides(cluster.GetCollectorImage(), flavor.CollectorRegistry, flavor.CollectorImageName, "")
	collectorImage := imageSpecFromOverrides(collectorImageOverrides)

	dynAdmissionControllerCfg := cluster.GetDynamicConfig().GetAdmissionControllerConfig()

	admissionControllerCfg := map[string]interface{}{
		"image": mainImage,
	}
	if features.AdmissionControllerConfig.Enabled() {
		admissionControllerCfg["dynamic"] = map[string]interface{}{
			"disableBypass": dynAdmissionControllerCfg.GetDisableBypass(),
			"enforce":       dynAdmissionControllerCfg.GetEnabled() || dynAdmissionControllerCfg.GetEnforceOnUpdates(),
		}
		admissionControllerCfg["failurePolicy"] = "Ignore"
		if cluster.GetAdmissionControllerFailOnError() {
			admissionControllerCfg["failurePolicy"] = "Fail"
		}
	} else {
		admissionControllerCfg["listenOnCreates"] = cluster.GetAdmissionController()
		admissionControllerCfg["listenOnUpdates"] = cluster.GetAdmissionControllerUpdates()
		admissionControllerCfg["listenOnEvents"] = cluster.GetAdmissionControllerEvents()
		admissionControllerCfg["dynamic"] = map[string]interface{}{
			"enforceOnCreates": dynAdmissionControllerCfg.GetEnabled(),
			"enforceOnUpdates": dynAdmissionControllerCfg.GetEnforceOnUpdates(),
			"scanInline":       dynAdmissionControllerCfg.GetScanInline(),
			"disableBypass":    dynAdmissionControllerCfg.GetDisableBypass(),
			"timeout":          float64(dynAdmissionControllerCfg.GetTimeoutSeconds()),
		}
	}

	m := map[string]interface{}{
		"clusterName":     cluster.GetName(),
		"centralEndpoint": cluster.GetCentralApiEndpoint(),
		"sensor": map[string]interface{}{
			"image": mainImage,
		},
		"admissionControl": admissionControllerCfg,
		"collector": map[string]interface{}{
			"collectionMethod":        cluster.GetCollectionMethod().String(),
			"disableTaintTolerations": cluster.GetTolerationsConfig().GetDisabled(),
			"slimMode":                cluster.GetSlimCollector(),
			"image":                   collectorImage,
		},
	}

	return maputil.NormalizeGenericMap(m), nil
}
