package types

import "github.com/stackrox/rox/generated/storage"

// ConvertDeploymentToDeploymentList converts a storage.Deployment to a storage.ListDeployment
func ConvertDeploymentToDeploymentList(d *storage.Deployment) *storage.ListDeployment {
	ld := &storage.ListDeployment{}
	ld.SetId(d.GetId())
	ld.SetHash(d.GetHash())
	ld.SetName(d.GetName())
	ld.SetCluster(d.GetClusterName())
	ld.SetClusterId(d.GetClusterId())
	ld.SetNamespace(d.GetNamespace())
	ld.SetCreated(d.GetCreated())
	ld.SetPriority(d.GetPriority())
	return ld
}
