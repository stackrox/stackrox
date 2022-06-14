package types

import "github.com/stackrox/rox/generated/storage"

// ConvertDeploymentToDeploymentList converts a storage.Deployment to a storage.ListDeployment
func ConvertDeploymentToDeploymentList(d *storage.Deployment) *storage.ListDeployment {
	return &storage.ListDeployment{
		Id:        d.GetId(),
		Hash:      d.GetHash(),
		Name:      d.GetName(),
		Cluster:   d.GetClusterName(),
		ClusterId: d.GetClusterId(),
		Namespace: d.GetNamespace(),
		Created:   d.GetCreated(),
		Priority:  d.GetPriority(),
	}
}
