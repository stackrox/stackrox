package framework

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/complianceoperator/api/v1alpha1"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
)

// ComplianceDomain is the domain (i.e., the set of all potential target objects) for a compliance run.
type ComplianceDomain interface {
	ID() string
	Cluster() ComplianceTarget
	Nodes() []ComplianceTarget
	Deployments() []ComplianceTarget
	Pods() []*storage.Pod
	MachineConfigs() []ComplianceTarget
}

type complianceDomain struct {
	domainID       string
	cluster        clusterTarget
	nodes          []nodeTarget
	deployments    []deploymentTarget
	pods           []*storage.Pod
	machineConfigs []machineConfigTarget
}

func newComplianceDomain(cluster *storage.Cluster, nodes []*storage.Node, deployments []*storage.Deployment, pods []*storage.Pod, results []*storage.ComplianceOperatorCheckResult) *complianceDomain {
	clusterTarget := targetForCluster(cluster)
	nodeTargets := make([]nodeTarget, len(nodes))
	for i, node := range nodes {
		nodeTargets[i] = targetForNode(node)
	}
	deploymentTargets := make([]deploymentTarget, len(deployments))
	for i, deployment := range deployments {
		deploymentTargets[i] = targetForDeployment(deployment)
	}
	var machineConfigTargets []machineConfigTarget
	machineConfigs := set.NewStringSet()
	for _, r := range results {
		conf, ok := r.Labels[v1alpha1.ComplianceScanLabel]
		if ok && machineConfigs.Add(conf) {
			machineConfigTargets = append(machineConfigTargets, targetForMachineConfig(conf))
		}
	}

	return &complianceDomain{
		domainID:       uuid.NewV4().String(),
		cluster:        clusterTarget,
		nodes:          nodeTargets,
		deployments:    deploymentTargets,
		pods:           pods,
		machineConfigs: machineConfigTargets,
	}
}

func (d *complianceDomain) ID() string {
	return d.domainID
}

func (d *complianceDomain) Cluster() ComplianceTarget {
	return d.cluster
}

func (d *complianceDomain) Nodes() []ComplianceTarget {
	result := make([]ComplianceTarget, len(d.nodes))
	for i, node := range d.nodes {
		result[i] = node
	}
	return result
}

func (d *complianceDomain) Deployments() []ComplianceTarget {
	result := make([]ComplianceTarget, len(d.deployments))
	for i, deployment := range d.deployments {
		result[i] = deployment
	}
	return result
}

func (d *complianceDomain) MachineConfigs() []ComplianceTarget {
	result := make([]ComplianceTarget, len(d.machineConfigs))
	for i, mc := range d.machineConfigs {
		result[i] = mc
	}
	return result
}

func (d *complianceDomain) Pods() []*storage.Pod {
	result := make([]*storage.Pod, len(d.pods))
	for i, pod := range d.pods {
		result[i] = pod
	}
	return result
}

// NewComplianceDomain creates a new compliance domain from the given cluster, list of nodes and list of deployments.
func NewComplianceDomain(cluster *storage.Cluster, nodes []*storage.Node, deployments []*storage.Deployment, pods []*storage.Pod, results []*storage.ComplianceOperatorCheckResult) ComplianceDomain {
	return newComplianceDomain(cluster, nodes, deployments, pods, results)
}
