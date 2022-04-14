// Package resources lists all resource types used by Central.
package resources

import (
	"sort"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/features"
)

// All resource types that we want to define (for the purposes of enforcing
// API permissions) must be defined here.
//
// Description for each type and the meaning of the respective Read and Write
// operations is available in
//     "ui/apps/platform/src/Containers/AccessControl/PermissionSets/ResourceDescription.tsx"
//
// UI defines possible values for resource type in
//     "ui/apps/platform/src/types/roleResources.ts"
//
// Each time you touch the list below, you likely need to update both
// aforementioned files.
//
// KEEP THE FOLLOWING LIST SORTED IN LEXICOGRAPHIC ORDER (case-sensitive).
var (
	APIToken                         = newResourceMetadata("APIToken", permissions.GlobalScope)
	Alert                            = newResourceMetadata("Alert", permissions.NamespaceScope)
	AllComments                      = newResourceMetadata("AllComments", permissions.GlobalScope)
	AuthPlugin                       = newResourceMetadata("AuthPlugin", permissions.GlobalScope)
	AuthProvider                     = newResourceMetadata("AuthProvider", permissions.GlobalScope)
	BackupPlugins                    = newResourceMetadata("BackupPlugins", permissions.GlobalScope)
	CVE                              = newResourceMetadata("CVE", permissions.NamespaceScope)
	Cluster                          = newResourceMetadata("Cluster", permissions.ClusterScope)
	Compliance                       = newResourceMetadata("Compliance", permissions.ClusterScope)
	ComplianceRunSchedule            = newResourceMetadata("ComplianceRunSchedule", permissions.GlobalScope)
	ComplianceRuns                   = newResourceMetadata("ComplianceRuns", permissions.ClusterScope)
	Config                           = newResourceMetadata("Config", permissions.GlobalScope)
	DebugLogs                        = newResourceMetadata("DebugLogs", permissions.GlobalScope)
	Deployment                       = newResourceMetadata("Deployment", permissions.NamespaceScope)
	Detection                        = newResourceMetadata("Detection", permissions.GlobalScope)
	Group                            = newResourceMetadata("Group", permissions.GlobalScope)
	Image                            = newResourceMetadata("Image", permissions.NamespaceScope)
	ImageComponent                   = newResourceMetadata("ImageComponent", permissions.NamespaceScope)
	ImageIntegration                 = newResourceMetadata("ImageIntegration", permissions.GlobalScope)
	Indicator                        = newResourceMetadata("Indicator", permissions.NamespaceScope)
	K8sRole                          = newResourceMetadata("K8sRole", permissions.NamespaceScope)
	K8sRoleBinding                   = newResourceMetadata("K8sRoleBinding", permissions.NamespaceScope)
	K8sSubject                       = newResourceMetadata("K8sSubject", permissions.NamespaceScope)
	Licenses                         = newResourceMetadata("Licenses", permissions.GlobalScope)
	Namespace                        = newResourceMetadata("Namespace", permissions.NamespaceScope)
	NetworkBaseline                  = newResourceMetadata("NetworkBaseline", permissions.NamespaceScope)
	NetworkGraph                     = newResourceMetadata("NetworkGraph", permissions.NamespaceScope)
	NetworkGraphConfig               = newResourceMetadata("NetworkGraphConfig", permissions.GlobalScope)
	NetworkPolicy                    = newResourceMetadata("NetworkPolicy", permissions.NamespaceScope)
	Node                             = newResourceMetadata("Node", permissions.ClusterScope)
	Notifier                         = newResourceMetadata("Notifier", permissions.GlobalScope)
	Policy                           = newResourceMetadata("Policy", permissions.GlobalScope)
	ProbeUpload                      = newResourceMetadata("ProbeUpload", permissions.GlobalScope)
	ProcessWhitelist                 = newResourceMetadata("ProcessWhitelist", permissions.NamespaceScope)
	Risk                             = newResourceMetadata("Risk", permissions.NamespaceScope)
	Role                             = newResourceMetadata("Role", permissions.GlobalScope)
	ScannerBundle                    = newResourceMetadata("ScannerBundle", permissions.GlobalScope)
	ScannerDefinitions               = newResourceMetadata("ScannerDefinitions", permissions.GlobalScope)
	Secret                           = newResourceMetadata("Secret", permissions.NamespaceScope)
	SensorUpgradeConfig              = newResourceMetadata("SensorUpgradeConfig", permissions.GlobalScope)
	ServiceAccount                   = newResourceMetadata("ServiceAccount", permissions.NamespaceScope)
	ServiceIdentity                  = newResourceMetadata("ServiceIdentity", permissions.GlobalScope)
	SignatureIntegration             = newResourceMetadataWithFeatureFlag("SignatureIntegration", permissions.GlobalScope, features.ImageSignatureVerification)
	User                             = newResourceMetadata("User", permissions.GlobalScope)
	VulnerabilityManagementApprovals = newResourceMetadata("VulnerabilityManagementApprovals", permissions.GlobalScope)
	VulnerabilityManagementRequests  = newResourceMetadata("VulnerabilityManagementRequests", permissions.GlobalScope)
	VulnerabilityReports             = newResourceMetadataWithFeatureFlag("VulnerabilityReports", permissions.GlobalScope, features.VulnReporting)
	WatchedImage                     = newResourceMetadata("WatchedImage", permissions.GlobalScope)

	// Internal Resources
	ComplianceOperator = newInternalResourceMetadata("ComplianceOperator", permissions.GlobalScope)

	resourceToMetadata         = make(map[permissions.Resource]permissions.ResourceMetadata)
	disabledResourceToMetadata = make(map[permissions.Resource]permissions.ResourceMetadata)
)

func newResourceMetadata(name permissions.Resource, scope permissions.ResourceScope) permissions.ResourceMetadata {
	md := permissions.ResourceMetadata{
		Resource: name,
		Scope:    scope,
	}
	resourceToMetadata[name] = md
	return md
}

func newResourceMetadataWithFeatureFlag(name permissions.Resource, scope permissions.ResourceScope, flag features.FeatureFlag) permissions.ResourceMetadata {
	md := permissions.ResourceMetadata{
		Resource: name,
		Scope:    scope,
	}
	if flag.Enabled() {
		resourceToMetadata[name] = md
	} else {
		disabledResourceToMetadata[name] = md
	}
	return md
}

func newInternalResourceMetadata(name permissions.Resource, scope permissions.ResourceScope) permissions.ResourceMetadata {
	return permissions.ResourceMetadata{
		Resource: name,
		Scope:    scope,
	}
}

// ListAll returns a list of all resources.
func ListAll() []permissions.Resource {
	resources := make([]permissions.Resource, 0, len(resourceToMetadata))
	for _, metadata := range ListAllMetadata() {
		resources = append(resources, metadata.Resource)
	}
	return resources
}

// ListAllMetadata returns a list of all resource metadata.
func ListAllMetadata() []permissions.ResourceMetadata {
	metadatas := make([]permissions.ResourceMetadata, 0, len(resourceToMetadata))
	for _, metadata := range resourceToMetadata {
		metadatas = append(metadatas, metadata)
	}
	sort.SliceStable(metadatas, func(i, j int) bool {
		return string(metadatas[i].Resource) < string(metadatas[j].Resource)
	})
	return metadatas
}

// ListAllDisabledMetadata returns a list of all resource metadata that are currently disable by feature flag.
func ListAllDisabledMetadata() []permissions.ResourceMetadata {
	metadatas := make([]permissions.ResourceMetadata, 0, len(disabledResourceToMetadata))
	for _, metadata := range disabledResourceToMetadata {
		metadatas = append(metadatas, metadata)
	}
	sort.SliceStable(metadatas, func(i, j int) bool {
		return string(metadatas[i].Resource) < string(metadatas[j].Resource)
	})
	return metadatas
}

// AllResourcesViewPermissions returns a slice containing view permissions for all resource types.
func AllResourcesViewPermissions() []permissions.ResourceWithAccess {
	metadatas := ListAllMetadata()
	result := make([]permissions.ResourceWithAccess, len(metadatas))
	for i, metadata := range metadatas {
		result[i] = permissions.ResourceWithAccess{
			// We want to ensure access to *all* resources, so when using SAC, always perform legacy auth (= enforcement
			// at the global scope) even for cluster- or namespace-scoped resources.
			Resource: permissions.WithLegacyAuthForSAC(metadata, true),
			Access:   storage.Access_READ_ACCESS,
		}
	}
	return result
}

// AllResourcesModifyPermissions returns a slice containing write permissions for all resource types.
func AllResourcesModifyPermissions() []permissions.ResourceWithAccess {
	metadatas := ListAllMetadata()
	result := make([]permissions.ResourceWithAccess, len(metadatas))
	for i, metadata := range metadatas {
		result[i] = permissions.ResourceWithAccess{
			// We want to ensure access to *all* resources, so when using SAC, always perform legacy auth (= enforcement
			// at the global scope) even for cluster- or namespace-scoped resources.
			Resource: permissions.WithLegacyAuthForSAC(metadata, true),
			Access:   storage.Access_READ_WRITE_ACCESS,
		}
	}
	return result
}

// MetadataForResource returns the metadata for the given resource. If the resource is unknown, metadata for this
// resource with global scope is returned.
func MetadataForResource(res permissions.Resource) (permissions.ResourceMetadata, bool) {
	md, found := resourceToMetadata[res]
	if !found {
		md.Resource = res
		md.Scope = permissions.GlobalScope
	}
	return md, found
}
