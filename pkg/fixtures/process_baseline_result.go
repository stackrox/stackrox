package fixtures

import "github.com/stackrox/rox/generated/storage"

// GetScopedProcessBaselineResult returns a mock ProcessBaselineResult belonging to the input scope.
func GetScopedProcessBaselineResult(id string, clusterID string, namespace string) *storage.ProcessBaselineResults {
	pbr := &storage.ProcessBaselineResults{}
	pbr.SetDeploymentId(id)
	pbr.SetClusterId(clusterID)
	pbr.SetNamespace(namespace)
	return pbr
}
