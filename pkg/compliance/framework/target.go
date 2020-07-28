package framework

import "fmt"

// TargetKind indicates the kind of a compliance check target (cluster, node, deployment).
type TargetKind int

const (
	// ClusterKind indicates that a compliance check target is of type Cluster.
	ClusterKind TargetKind = iota
	// NodeKind indicates that a compliance check target is of type Node.
	NodeKind
	// DeploymentKind indicates that a compliance check target is of type Deployment.
	DeploymentKind
)

func (k TargetKind) String() string {
	switch k {
	case ClusterKind:
		return "cluster"
	case NodeKind:
		return "node"
	case DeploymentKind:
		return "deployment"
	default:
		return fmt.Sprintf("TargetKind(%d)", int(k))
	}
}
