package objects

import "github.com/stackrox/stackrox/generated/storage"

// ToListDeployment converts a deployment to a list deployment.
func ToListDeployment(d *storage.Deployment) *storage.ListDeployment {
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

// DeploymentsMapByID converts the given Deployment slice into a map indexed by the deployment ID.
func DeploymentsMapByID(deployments []*storage.Deployment) map[string]*storage.Deployment {
	result := make(map[string]*storage.Deployment)
	for _, deployment := range deployments {
		result[deployment.GetId()] = deployment
	}
	return result
}

// ListDeploymentsMapByID converts the given ListDeployment slice into a map indexed by the deployment ID.
func ListDeploymentsMapByID(deployments []*storage.ListDeployment) map[string]*storage.ListDeployment {
	result := make(map[string]*storage.ListDeployment)
	for _, deployment := range deployments {
		result[deployment.GetId()] = deployment
	}
	return result
}

// ListDeploymentsMapByIDFromDeployments converts the given Deployment slice into a ListDeployment map indexed by the
// deployment ID.
func ListDeploymentsMapByIDFromDeployments(deployments []*storage.Deployment) map[string]*storage.ListDeployment {
	result := make(map[string]*storage.ListDeployment)
	for _, deployment := range deployments {
		result[deployment.GetId()] = ToListDeployment(deployment)
	}
	return result
}
