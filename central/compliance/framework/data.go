package framework

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

//go:generate mockgen-wrapper ComplianceDataRepository

// ComplianceDataRepository is the unified interface for accessing all the data that might be relevant for a compliance
// run. This provides check implementors with a unified view of all data objects regardless of their source (stored by
// central vs. obtained specifically for a compliance run), and also allows presenting a stable snapshot to all checks
// to reduce the risk of inconsistencies.
type ComplianceDataRepository interface {
	Cluster() *storage.Cluster
	Nodes() map[string]*storage.Node
	Deployments() map[string]*storage.Deployment

	Alerts() []*storage.ListAlert
	NetworkPolicies() map[string]*storage.NetworkPolicy
	NetworkGraph() *v1.NetworkGraph
	Policies() map[string]*storage.Policy
	ImageIntegrations() []*storage.ImageIntegration
	ProcessIndicators() []*storage.ProcessIndicator
	NetworkFlows() []*storage.NetworkFlow
	PolicyCategories() map[string]set.StringSet

	// Per-host data

	HostFiles(node *storage.Node) map[string]*compliance.File
	HostProcesses(node *storage.Node) []*compliance.CommandLine
}
