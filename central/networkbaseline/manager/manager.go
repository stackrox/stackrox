package manager

import (
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/timestamp"
)

//go:generate mockgen-wrapper
// The Manager manages network baselines.
// ALL writes to network baselines MUST go through the manager.
type Manager interface {
	// ProcessDeploymentCreate notifies the baseline manager of a deployment create.
	// The baseline manager then creates a baseline for this deployment if it does not already exist.
	// It must only be called by trusted code, since it assumes the caller has full access to modify
	// network baselines in the datastore.
	ProcessDeploymentCreate(deploymentID, clusterID, namespace string) error
	// ProcessFlowUpdate notifies the baseline manager of a dump of a batch of network flows.
	// It must only be called by trusted code, since it assumes the caller has full access to modify
	// network baselines in the datastore.
	ProcessFlowUpdate(flows map[networkgraph.NetworkConnIndicator]timestamp.MicroTS) error
}
