package objects

import (
	"github.com/stackrox/rox/generated/storage"
	"google.golang.org/protobuf/proto"
)

// ToListDeployment converts a deployment to a list deployment.
func ToListDeployment(d *storage.Deployment) *storage.ListDeployment {
	return storage.ListDeployment_builder{
		Id:        proto.String(d.GetId()),
		Hash:      proto.Uint64(d.GetHash()),
		Name:      proto.String(d.GetName()),
		Cluster:   proto.String(d.GetClusterName()),
		ClusterId: proto.String(d.GetClusterId()),
		Namespace: proto.String(d.GetNamespace()),
		Created:   d.GetCreated(),
		Priority:  &d.GetPriority(),
	}.Build()
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
