// Package resources lists all resource types used by Central.
package resources

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// All resource types that we want to define (for the purposes of enforcing
// API permissions) must be defined here.
var (
	APIToken              = newResource("APIToken")
	Alert                 = newResource("Alert")
	AuthProvider          = newResource("AuthProvider")
	BackupPlugins         = newResource("BackupPlugins")
	ClientTrustCerts      = newResource("ClientTrustCerts")
	Cluster               = newResource("Cluster")
	Compliance            = newResource("Compliance")
	ComplianceRunSchedule = newResource("ComplianceRunSchedule")
	ComplianceRuns        = newResource("ComplianceRuns")
	Config                = newResource("Config")
	DebugMetrics          = newResource("DebugMetrics")
	DebugLogs             = newResource("DebugLogs")
	Deployment            = newResource("Deployment")
	Detection             = newResource("Detection")
	Group                 = newResource("Group")
	Image                 = newResource("Image")
	ImageIntegration      = newResource("ImageIntegration")
	ImbuedLogs            = newResource("ImbuedLogs")
	Indicator             = newResource("Indicator")
	Licenses              = newResource("Licenses")
	Namespace             = newResource("Namespace")
	Node                  = newResource("Node")
	Notifier              = newResource("Notifier")
	NetworkPolicy         = newResource("NetworkPolicy")
	NetworkGraph          = newResource("NetworkGraph")
	Policy                = newResource("Policy")
	ProcessWhitelist      = newResource("ProcessWhitelist")
	Role                  = newResource("Role")
	Secret                = newResource("Secret")
	ServiceAccount        = newResource("ServiceAccount")
	ServiceIdentity       = newResource("ServiceIdentity")
	User                  = newResource("User")
	K8sRole               = newResource("K8sRole")
	K8sRoleBinding        = newResource("K8sRoleBinding")
	K8sSubject            = newResource("K8sSubject")

	allResources = permissions.NewResourceSet()
)

func newResource(name permissions.Resource) permissions.Resource {
	allResources.Add(name)
	return name
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
