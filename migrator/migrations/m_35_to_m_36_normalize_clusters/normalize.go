package m35tom36

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
)

const (
	defaultAdmissionControllerTimeout = 3
)

func needsNormalization(cluster *storage.Cluster) bool {
	return cluster.GetCollectionMethod() == storage.CollectionMethod_UNSET_COLLECTION || cluster.GetDynamicConfig().GetAdmissionControllerConfig() == nil
}

// normalizeCluster applies a transformation to cluster that ensures all the default settings enforcements that were
// (a) present in 3.0.42.0, (b) added after we started supporting database updates (this excludes the central API
// endpoint format/http(s) prefixes), and (c) not covered in a past migration (such as the tolerations config).
func normalizeCluster(cluster *storage.Cluster) {
	// For backwards compatibility reasons, if Collection Method is not set then honor defaults for runtime support
	if cluster.GetCollectionMethod() == storage.CollectionMethod_UNSET_COLLECTION {
		if !cluster.GetRuntimeSupport() {
			cluster.CollectionMethod = storage.CollectionMethod_NO_COLLECTION
		} else {
			cluster.CollectionMethod = storage.CollectionMethod_KERNEL_MODULE
		}
	}

	if cluster.GetDynamicConfig() == nil {
		cluster.DynamicConfig = &storage.DynamicClusterConfig{}
	}

	acConfig := cluster.DynamicConfig.GetAdmissionControllerConfig()
	if acConfig == nil {
		acConfig = &storage.AdmissionControllerConfig{
			Enabled: false,
		}
		cluster.DynamicConfig.AdmissionControllerConfig = acConfig
	}

	if acConfig.GetTimeoutSeconds() < 0 {
		acConfig.TimeoutSeconds = 0
		log.WriteToStderrf("Setting defaults for cluster %s: admission controller timeout of %d is invalid; applying default value", cluster.GetName(), acConfig.GetTimeoutSeconds())
	}

	if acConfig.GetTimeoutSeconds() == 0 {
		acConfig.TimeoutSeconds = defaultAdmissionControllerTimeout
	}
}
