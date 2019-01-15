package data

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

type repository struct {
	cluster     *storage.Cluster
	nodes       map[string]*storage.Node
	deployments map[string]*storage.Deployment

	networkPolicies   map[string]*storage.NetworkPolicy
	networkGraph      *v1.NetworkGraph
	policies          map[string]*storage.Policy
	imageIntegrations []*storage.ImageIntegration
}

func (r *repository) Cluster() *storage.Cluster {
	return r.cluster
}

func (r *repository) Nodes() map[string]*storage.Node {
	return r.nodes
}

func (r *repository) Deployments() map[string]*storage.Deployment {
	return r.deployments
}

func (r *repository) NetworkPolicies() map[string]*storage.NetworkPolicy {
	return r.networkPolicies
}

func (r *repository) NetworkGraph() *v1.NetworkGraph {
	return r.networkGraph
}

func (r *repository) Policies() map[string]*storage.Policy {
	return r.policies
}

func (r *repository) ImageIntegrations() []*storage.ImageIntegration {
	return r.imageIntegrations
}

func newRepository(domain framework.ComplianceDomain, factory *factory) (*repository, error) {
	r := &repository{}
	if err := r.init(domain, factory); err != nil {
		return nil, err
	}
	return r, nil
}

func nodesByID(nodes []*storage.Node) map[string]*storage.Node {
	result := make(map[string]*storage.Node, len(nodes))
	for _, node := range nodes {
		result[node.GetId()] = node
	}
	return result
}

func deploymentsByID(deployments []*storage.Deployment) map[string]*storage.Deployment {
	result := make(map[string]*storage.Deployment, len(deployments))
	for _, deployment := range deployments {
		result[deployment.GetId()] = deployment
	}
	return result
}

func networkPoliciesByID(policies []*storage.NetworkPolicy) map[string]*storage.NetworkPolicy {
	result := make(map[string]*storage.NetworkPolicy, len(policies))
	for _, policy := range policies {
		result[policy.GetId()] = policy
	}
	return result
}

func policiesByName(policies []*storage.Policy) map[string]*storage.Policy {
	result := make(map[string]*storage.Policy, len(policies))
	for _, policy := range policies {
		result[policy.GetName()] = policy
	}
	return result
}

func (r *repository) init(domain framework.ComplianceDomain, f *factory) error {
	r.cluster = domain.Cluster().Cluster()
	r.nodes = nodesByID(framework.Nodes(domain))

	deployments := framework.Deployments(domain)
	r.deployments = deploymentsByID(deployments)

	clusterID := r.cluster.GetId()
	networkPolicies, err := f.networkPoliciesStore.GetNetworkPolicies(&v1.GetNetworkPoliciesRequest{
		ClusterId: clusterID,
	})
	if err != nil {
		return err
	}
	r.networkPolicies = networkPoliciesByID(networkPolicies)

	r.networkGraph = f.networkGraphEvaluator.GetGraph(deployments, networkPolicies)

	policies, err := f.policyStore.GetPolicies()
	if err != nil {
		return err
	}
	r.policies = policiesByName(policies)

	r.imageIntegrations, err = f.imageIntegrationStore.GetImageIntegrations(
		&v1.GetImageIntegrationsRequest{},
	)
	if err != nil {
		return err
	}
	return nil
}
