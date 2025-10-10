package types

import "github.com/stackrox/rox/generated/storage"

// ConvertDeploymentToDeploymentList converts a storage.Deployment to a storage.ListDeployment
func ConvertDeploymentToDeploymentList(d *storage.Deployment) *storage.ListDeployment {
	id := d.GetId()
	hash := d.GetHash()
	name := d.GetName()
	cluster := d.GetClusterName()
	clusterId := d.GetClusterId()
	namespace := d.GetNamespace()
	priority := d.GetPriority()
	return storage.ListDeployment_builder{
		Id:        &id,
		Hash:      &hash,
		Name:      &name,
		Cluster:   &cluster,
		ClusterId: &clusterId,
		Namespace: &namespace,
		Created:   d.GetCreated(),
		Priority:  &priority,
	}.Build()
}
