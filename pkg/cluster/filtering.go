package cluster

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

const (
	openshiftFilter = "openshift-.*"
	noPersistence   = ".*"
)

func GetNamespaceFilter(cluster *storage.Cluster) string {
	var config *storage.DynamicClusterConfig_RuntimeDataControl

	if cluster.GetManagedBy() == storage.ManagerType_MANAGER_TYPE_MANUAL || cluster.GetManagedBy() == storage.ManagerType_MANAGER_TYPE_UNKNOWN {
		config = cluster.GetDynamicConfig().GetRuntimeDataControl()
	} else {
		config = cluster.GetHelmConfig().GetDynamicConfig().GetRuntimeDataControl()
	}

	// Persistence filter has highest priority, since any other
	// option is only its subset
	if !config.GetPersistence() {
		return noPersistence
	}

	filter := config.GetNamespaceFilter()

	if config.GetExcludeOpenshift() {
		// Openshift exclusion filter could be combined with the custom filter
		if filter != "" {
			filter = fmt.Sprintf("%s|%s", filter, openshiftFilter)
		} else {
			filter = openshiftFilter
		}
	}

	return filter
}
