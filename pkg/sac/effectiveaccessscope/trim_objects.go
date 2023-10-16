package effectiveaccessscope

import (
	"github.com/stackrox/rox/generated/storage"
)

// TrimCluster removes all attributes from the input object, except the ones
// that are used in effective access scope computation and the raw type ones.
func TrimCluster(cluster *storage.Cluster) {
	cluster.MainImage = ""
	cluster.CollectorImage = ""
	cluster.CentralApiEndpoint = ""
	cluster.Status = nil
	cluster.DynamicConfig = nil
	cluster.TolerationsConfig = nil
	cluster.HealthStatus = nil
	cluster.HelmConfig = nil
	cluster.MostRecentSensorId = nil
	cluster.AuditLogState = nil
	cluster.InitBundleId = ""
}

// TrimNamespace removes all attributes from the input object, except the ones
// that are used in effective access scope computation and the raw type ones.
func TrimNamespace(namespace *storage.NamespaceMetadata) {
	namespace.ClusterId = ""
	namespace.CreationTime = nil
	namespace.Annotations = nil
}
