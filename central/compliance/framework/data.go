package framework

import (
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

//go:generate mockgen-wrapper

// ImageMatcher is the sub-interface of our integration objects that is
// relevant to the ComplianceDataRepository.
type ImageMatcher interface {
	Match(image *storage.ImageName) bool
}

// ComplianceDataRepository is the unified interface for accessing all the data that might be relevant for a compliance
// run. This provides check implementors with a unified view of all data objects regardless of their source (stored by
// central vs. obtained specifically for a compliance run), and also allows presenting a stable snapshot to all checks
// to reduce the risk of inconsistencies.
type ComplianceDataRepository interface {
	Cluster() *storage.Cluster
	Nodes() map[string]*storage.Node
	Deployments() map[string]*storage.Deployment

	UnresolvedAlerts() []*storage.ListAlert
	NetworkPolicies() map[string]*storage.NetworkPolicy
	DeploymentsToNetworkPolicies() map[string][]*storage.NetworkPolicy
	// Policies returns all policies, keyed by their name.
	Policies() map[string]*storage.Policy
	Images() []*storage.ListImage
	ImageIntegrations() []*storage.ImageIntegration
	RegistryIntegrations() []ImageMatcher
	ScannerIntegrations() []ImageMatcher
	SSHProcessIndicators() []*storage.ProcessIndicator
	HasProcessIndicators() bool
	NetworkFlowsWithDeploymentDst() []*storage.NetworkFlow
	PolicyCategories() map[string]set.StringSet
	Notifiers() []*storage.Notifier
	K8sRoles() []*storage.K8SRole
	K8sRoleBindings() []*storage.K8SRoleBinding
	CISKubernetesTriggered() bool

	ComplianceOperatorResults() map[string][]*storage.ComplianceOperatorCheckResult

	// Per-host data
	HostScraped(node *storage.Node) *compliance.ComplianceReturn
	NodeResults() map[string]map[string]*compliance.ComplianceStandardResult

	AddHostScrapedData(scrapeResults map[string]*compliance.ComplianceReturn)
}
