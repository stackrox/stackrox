package framework

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/framework"
)

// ComplianceTarget is the target for a check (cluster, node, or deployment).
type ComplianceTarget interface {
	Kind() framework.TargetKind

	// ID returns the ID of this target. For a given compliance domain, the combination of kind and ID uniquely
	// identifies an object.
	ID() string

	// The following methods allow obtaining the actual target object. Calling a method that does not correspond
	// to the respective target kind (as returned by `Kind()`) will result in a panic.

	Cluster() *storage.Cluster
	Node() *storage.Node
	Deployment() *storage.Deployment
	MachineConfig() string
}

// TargetRef is an identifier for a compliance target that can be used as a map key and compared for equality.
type TargetRef struct {
	Kind framework.TargetKind
	ID   string
}

// GetTargetRef obtains a TargetRef for a compliance target.
func GetTargetRef(target ComplianceTarget) TargetRef {
	return TargetRef{
		Kind: target.Kind(),
		ID:   target.ID(),
	}
}

// TargetObject obtains the underlying object for a target.
func TargetObject(tgt ComplianceTarget) interface{} {
	switch tgt.Kind() {
	case framework.ClusterKind:
		return tgt.Cluster()
	case framework.NodeKind:
		return tgt.Node()
	case framework.DeploymentKind:
		return tgt.Deployment()
	default:
		panic(fmt.Errorf("unknown target kind %v", tgt.Kind()))
	}
}

type baseTarget struct {
	kind framework.TargetKind
}

func (t baseTarget) Cluster() *storage.Cluster {
	panic(fmt.Errorf("requested cluster target, but target kind of active scope is %v", t.kind))
}

func (t baseTarget) Node() *storage.Node {
	panic(fmt.Errorf("requested node target, but target kind of active scope is %v", t.kind))
}

func (t baseTarget) Deployment() *storage.Deployment {
	panic(fmt.Errorf("requested deployment target, but target kind of active scope is %v", t.kind))
}

func (t baseTarget) MachineConfig() string {
	panic(fmt.Errorf("requested machine config target, but target kind of active scope is %v", t.kind))
}

func (t baseTarget) Kind() framework.TargetKind {
	return t.kind
}

type clusterTarget struct {
	baseTarget
	cluster *storage.Cluster
}

func (t clusterTarget) ID() string {
	return t.cluster.GetId()
}
func (t clusterTarget) Cluster() *storage.Cluster {
	return t.cluster
}

type nodeTarget struct {
	baseTarget
	node *storage.Node
}

func (t nodeTarget) ID() string {
	return t.node.GetId()
}
func (t nodeTarget) Node() *storage.Node {
	return t.node
}

type deploymentTarget struct {
	baseTarget
	deployment *storage.Deployment
}

func (t deploymentTarget) ID() string {
	return t.deployment.GetId()
}
func (t deploymentTarget) Deployment() *storage.Deployment {
	return t.deployment
}

type machineConfigTarget struct {
	baseTarget
	name string
}

func (t machineConfigTarget) ID() string {
	return t.name
}
func (t machineConfigTarget) MachineConfig() string {
	return t.name
}

func targetForCluster(cluster *storage.Cluster) clusterTarget {
	return clusterTarget{
		baseTarget: baseTarget{
			kind: framework.ClusterKind,
		},
		cluster: cluster,
	}
}

func targetForNode(node *storage.Node) nodeTarget {
	return nodeTarget{
		baseTarget: baseTarget{
			kind: framework.NodeKind,
		},
		node: node,
	}
}

func targetForDeployment(deployment *storage.Deployment) deploymentTarget {
	return deploymentTarget{
		baseTarget: baseTarget{
			kind: framework.DeploymentKind,
		},
		deployment: deployment,
	}
}

func targetForMachineConfig(name string) machineConfigTarget {
	return machineConfigTarget{
		baseTarget: baseTarget{
			kind: framework.MachineConfigKind,
		},
		name: name,
	}
}
