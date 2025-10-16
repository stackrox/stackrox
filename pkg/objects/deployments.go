package objects

import "github.com/stackrox/rox/generated/storage"

// ToListDeployment converts a deployment to a list deployment.
func ToListDeployment(d *storage.Deployment) *storage.ListDeployment {
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

// ListDeploymentsMapByID converts the given ListDeployment slice into a map indexed by the deployment ID.
func ListDeploymentsMapByID(deployments []*storage.ListDeployment) map[string]*storage.ListDeployment {
	result := make(map[string]*storage.ListDeployment, len(deployments))
	for _, deployment := range deployments {
		result[deployment.GetId()] = deployment
	}
	return result
}

// ListDeploymentsMapByIDFromDeployments converts the given Deployment slice into a ListDeployment map indexed by the
// deployment ID.
func ListDeploymentsMapByIDFromDeployments(deployments []*storage.Deployment) map[string]*storage.ListDeployment {
	result := make(map[string]*storage.ListDeployment, len(deployments))
	for _, deployment := range deployments {
		result[deployment.GetId()] = ToListDeployment(deployment)
	}
	return result
}
