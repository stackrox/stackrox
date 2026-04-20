package cluster

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/pointers"
)

const (
	openshiftFilter = "^openshift$|^openshift-.*"
	noPersistence   = ".*"
)

func GetNamespaceFilter(cluster *storage.Cluster) *string {
	var config *storage.DynamicClusterConfig_ProcessIndicatorsConfig

	if cluster.GetManagedBy() == storage.ManagerType_MANAGER_TYPE_MANUAL || cluster.GetManagedBy() == storage.ManagerType_MANAGER_TYPE_UNKNOWN {
		config = cluster.GetDynamicConfig().GetProcessIndicators()
	} else {
		config = cluster.GetHelmConfig().GetDynamicConfig().GetProcessIndicators()
	}

	// No configuration for runtime data means no filter
	if config == nil {
		return nil
	}

	// Persistence filter has highest priority, since any other
	// option is only its subset
	if config.GetNoPersistence() {
		return pointers.String(noPersistence)
	}

	filter := config.GetExcludeNamespaceFilter()

	if config.GetExcludeOpenshiftNs() {
		// Openshift exclusion filter could be combined with the custom filter
		if filter != "" {
			filter = fmt.Sprintf("%s|%s", filter, openshiftFilter)
		} else {
			filter = openshiftFilter
		}
	}

	// If everything is present, but empty, set no filter
	if filter == "" {
		return nil
	}

	return pointers.String(filter)
}
