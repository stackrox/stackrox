// Package resources lists all resource types used by Central.
package resources

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// All resource types that we want to define (for the purposes of enforcing
// API permissions) must be defined here.
var (
	APIToken              = newResourceMetadata("APIToken", permissions.GlobalScope)
	Alert                 = newResourceMetadata("Alert", permissions.NamespaceScope)
	AuthProvider          = newResourceMetadata("AuthProvider", permissions.GlobalScope)
	BackupPlugins         = newResourceMetadata("BackupPlugins", permissions.GlobalScope)
	ClientTrustCerts      = newResourceMetadata("ClientTrustCerts", permissions.GlobalScope)
	Cluster               = newResourceMetadata("Cluster", permissions.ClusterScope)
	Compliance            = newResourceMetadata("Compliance", permissions.GlobalScope)
	ComplianceRunSchedule = newResourceMetadata("ComplianceRunSchedule", permissions.GlobalScope)
	ComplianceRuns        = newResourceMetadata("ComplianceRuns", permissions.ClusterScope)
	Config                = newResourceMetadata("Config", permissions.GlobalScope)
	DebugMetrics          = newResourceMetadata("DebugMetrics", permissions.GlobalScope)
	DebugLogs             = newResourceMetadata("DebugLogs", permissions.GlobalScope)
	Deployment            = newResourceMetadata("Deployment", permissions.NamespaceScope)
	Detection             = newResourceMetadata("Detection", permissions.GlobalScope)
	Group                 = newResourceMetadata("Group", permissions.GlobalScope)
	Image                 = newResourceMetadata("Image", permissions.NamespaceScope)
	ImageIntegration      = newResourceMetadata("ImageIntegration", permissions.GlobalScope)
	ImbuedLogs            = newResourceMetadata("ImbuedLogs", permissions.GlobalScope)
	Indicator             = newResourceMetadata("Indicator", permissions.NamespaceScope)
	Licenses              = newResourceMetadata("Licenses", permissions.GlobalScope)
	Namespace             = newResourceMetadata("Namespace", permissions.NamespaceScope)
	Node                  = newResourceMetadata("Node", permissions.ClusterScope)
	Notifier              = newResourceMetadata("Notifier", permissions.GlobalScope)
	NetworkPolicy         = newResourceMetadata("NetworkPolicy", permissions.NamespaceScope)
	NetworkGraph          = newResourceMetadata("NetworkGraph", permissions.NamespaceScope)
	Policy                = newResourceMetadata("Policy", permissions.GlobalScope)
	ProcessWhitelist      = newResourceMetadata("ProcessWhitelist", permissions.NamespaceScope)
	Role                  = newResourceMetadata("Role", permissions.GlobalScope)
	Secret                = newResourceMetadata("Secret", permissions.NamespaceScope)
	ServiceAccount        = newResourceMetadata("ServiceAccount", permissions.NamespaceScope)
	ServiceIdentity       = newResourceMetadata("ServiceIdentity", permissions.GlobalScope)
	User                  = newResourceMetadata("User", permissions.GlobalScope)
	K8sRole               = newResourceMetadata("K8sRole", permissions.NamespaceScope)
	K8sRoleBinding        = newResourceMetadata("K8sRoleBinding", permissions.NamespaceScope)
	K8sSubject            = newResourceMetadata("K8sSubject", permissions.NamespaceScope)

	allResources = permissions.NewResourceSet()
)

func newResourceMetadata(name permissions.Resource, scope permissions.ResourceScope) permissions.ResourceMetadata {
	allResources.Add(name)
	return permissions.ResourceMetadata{
		Resource: name,
		Scope:    scope,
	}
}

// ListAll returns a list of all resources.
func ListAll() []permissions.Resource {
	return allResources.AsSortedSlice(func(i, j permissions.Resource) bool {
		return i < j
	})
}

// AllResourcesViewPermissions returns a slice containing view permissions for all resource types.
func AllResourcesViewPermissions() []*v1.Permission {
	resourceLst := ListAll()
	result := make([]*v1.Permission, len(resourceLst))
	for i, resource := range resourceLst {
		result[i] = permissions.View(resource)
	}
	return result
}

// AllResourcesModifyPermissions returns a slice containing view permissions for all resource types.
func AllResourcesModifyPermissions() []*v1.Permission {
	resourceLst := ListAll()
	result := make([]*v1.Permission, len(resourceLst))
	for i, resource := range resourceLst {
		result[i] = permissions.Modify(resource)
	}
	return result
}
