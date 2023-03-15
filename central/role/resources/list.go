// Package resources lists all resource types used by Central.
package resources

import (
	"sort"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// All resource types that we want to define (for the purposes of enforcing
// API permissions) must be defined here.
//
// Description for each type and the meaning of the respective Read and Write
// operations is available in
//
//	"ui/apps/platform/src/Containers/AccessControl/PermissionSets/ResourceDescription.tsx"
//
// UI defines possible values for resource type in
//
//	"ui/apps/platform/src/types/roleResources.ts"
//
// Each time you touch the list below, you likely need to update both
// aforementioned files.
//
// KEEP THE FOLLOWING LIST SORTED IN LEXICOGRAPHIC ORDER (case-sensitive).
var (
	// Access groups all access-related resources. It aims to cover
	// configuration for authentication and authorization. For instance,
	// it has replaced: AuthProvider, Group, Licenses, Role, User.
	Access = newResourceMetadata("Access", permissions.GlobalScope)

	// Administration groups all administration-like resources except those
	// related to authentication and authorization. It aims to cover platform
	// configuration. For instance, it has replaced: AllComments, Config,
	// DebugLogs, NetworkGraphConfig, ProbeUpload, ScannerBundle,
	// ScannerDefinitions, SensorUpgradeConfig, ServiceIdentity.
	Administration = newResourceMetadata("Administration", permissions.GlobalScope)

	Alert      = newResourceMetadata("Alert", permissions.NamespaceScope)
	CVE        = newResourceMetadata("CVE", permissions.NamespaceScope)
	Cluster    = newResourceMetadata("Cluster", permissions.ClusterScope)
	Compliance = newResourceMetadata("Compliance", permissions.ClusterScope)
	Deployment = newResourceMetadata("Deployment", permissions.NamespaceScope)

	// DeploymentExtension aims to cover our extensions to deployments. For
	// instance, it has replaced: Indicator, NetworkBaseline, ProcessWhitelist,
	// Risk.
	DeploymentExtension = newResourceMetadata("DeploymentExtension", permissions.NamespaceScope)

	Detection = newResourceMetadata("Detection", permissions.GlobalScope)
	Image     = newResourceMetadata("Image", permissions.NamespaceScope)

	// Integration groups all integration-related resources. It aims to cover
	// integrations and their configuration. For instance, it has replaced:
	// APIToken, BackupPlugins, ImageIntegration, Notifier, SignatureIntegration.
	Integration = newResourceMetadata("Integration", permissions.GlobalScope)

	K8sRole        = newResourceMetadata("K8sRole", permissions.NamespaceScope)
	K8sRoleBinding = newResourceMetadata("K8sRoleBinding", permissions.NamespaceScope)
	K8sSubject     = newResourceMetadata("K8sSubject", permissions.NamespaceScope)
	Namespace      = newResourceMetadata("Namespace", permissions.NamespaceScope)
	NetworkGraph   = newResourceMetadata("NetworkGraph", permissions.NamespaceScope)
	NetworkPolicy  = newResourceMetadata("NetworkPolicy", permissions.NamespaceScope)
	Node           = newResourceMetadata("Node", permissions.ClusterScope)

	Secret                           = newResourceMetadata("Secret", permissions.NamespaceScope)
	ServiceAccount                   = newResourceMetadata("ServiceAccount", permissions.NamespaceScope)
	VulnerabilityManagementApprovals = newResourceMetadata("VulnerabilityManagementApprovals",
		permissions.GlobalScope)
	VulnerabilityManagementRequests = newResourceMetadata("VulnerabilityManagementRequests",
		permissions.GlobalScope)

	WatchedImage = newResourceMetadata("WatchedImage", permissions.GlobalScope)
	// WorkflowAdministration groups all workflow-related resources. It aims to cover core workflows
	// such as managing policies and vulnerability reports. For instance, it has replaced:
	// Policy, VulnerabilityReports.
	WorkflowAdministration = newResourceMetadata("WorkflowAdministration", permissions.GlobalScope)

	// To-be-deprecated in 4.0 with ROX-12750 (deprecation notice in 3.73).
	AllComments = newDeprecatedResourceMetadata("AllComments", permissions.GlobalScope,
		Administration)
	// To-be-deprecated in 4.0 with ROX-12750 (deprecation notice in 3.73).
	ComplianceRuns = newDeprecatedResourceMetadata("ComplianceRuns", permissions.ClusterScope,
		Compliance)
	// To-be-deprecated in 4.0 with ROX-12750 (deprecation notice in 3.73).
	Config = newDeprecatedResourceMetadata("Config", permissions.GlobalScope,
		Administration)
	// To-be-deprecated in 4.0 with ROX-12750 (deprecation notice in 3.73).
	DebugLogs = newDeprecatedResourceMetadata("DebugLogs", permissions.GlobalScope,
		Administration)
	// To-be-deprecated in 4.0 with ROX-12750 (deprecation notice in 3.73).
	NetworkGraphConfig = newDeprecatedResourceMetadata("NetworkGraphConfig",
		permissions.GlobalScope, Administration)
	// To-be-deprecated in 4.1 with ROX-13888 (deprecation notice in 3.74).
	Policy = newDeprecatedResourceMetadata("Policy", permissions.GlobalScope, WorkflowAdministration)
	// To-be-deprecated in 4.0 with ROX-12750 (deprecation notice in 3.73).
	ProbeUpload = newDeprecatedResourceMetadata("ProbeUpload", permissions.GlobalScope,
		Administration)
	// To-be-deprecated in 4.1 with ROX-14398 (deprecation notice in 3.74).
	Role = newDeprecatedResourceMetadata("Role", permissions.GlobalScope, Access)
	// To-be-deprecated in 4.0 with ROX-12750 (deprecation notice in 3.73).
	ScannerBundle = newDeprecatedResourceMetadata("ScannerBundle",
		permissions.GlobalScope, Administration)
	// To-be-deprecated in 4.0 with ROX-12750 (deprecation notice in 3.73).
	ScannerDefinitions = newDeprecatedResourceMetadata("ScannerDefinitions",
		permissions.GlobalScope, Administration)
	// To-be-deprecated in 4.0 with ROX-12750 (deprecation notice in 3.73).
	SensorUpgradeConfig = newDeprecatedResourceMetadata("SensorUpgradeConfig",
		permissions.GlobalScope, Administration)
	// To-be-deprecated in 4.0 with ROX-12750 (deprecation notice in 3.73).
	ServiceIdentity = newDeprecatedResourceMetadata("ServiceIdentity",
		permissions.GlobalScope, Administration)
	// To-be-deprecated in 4.1 with ROX-13888 (deprecation notice in 3.74).
	VulnerabilityReports = newDeprecatedResourceMetadata("VulnerabilityReports", permissions.GlobalScope,
		WorkflowAdministration)

	// Internal Resources.
	ComplianceOperator = newInternalResourceMetadata("ComplianceOperator", permissions.GlobalScope)
	InstallationInfo   = newInternalResourceMetadata("InstallationInfo", permissions.GlobalScope)
	Notifications      = newInternalResourceMetadata("Notifications", permissions.GlobalScope)
	Version            = newInternalResourceMetadata("Version", permissions.GlobalScope)

	resourceToMetadata         = make(map[permissions.Resource]permissions.ResourceMetadata)
	disabledResourceToMetadata = make(map[permissions.Resource]permissions.ResourceMetadata)
	internalResourceToMetadata = make(map[permissions.Resource]permissions.ResourceMetadata)
)

func newResourceMetadata(name permissions.Resource, scope permissions.ResourceScope) permissions.ResourceMetadata {
	md := permissions.ResourceMetadata{
		Resource: name,
		Scope:    scope,
	}
	resourceToMetadata[name] = md
	return md
}

func newDeprecatedResourceMetadata(name permissions.Resource, scope permissions.ResourceScope,
	replacingResourceMD permissions.ResourceMetadata) permissions.ResourceMetadata {
	md := permissions.ResourceMetadata{
		Resource:          name,
		Scope:             scope,
		ReplacingResource: &replacingResourceMD,
	}
	resourceToMetadata[name] = md
	return md
}

/*
Commented for now, uncomment in case you need to register a resource guarded behind a feature flag.
func newResourceMetadataWithFeatureFlag(name permissions.Resource, scope permissions.ResourceScope,
	flag features.FeatureFlag) permissions.ResourceMetadata {
	md := permissions.ResourceMetadata{
		Resource:          name,
		Scope:             scope,
	}
	if flag.Enabled() {
		resourceToMetadata[name] = md
	} else {
		disabledResourceToMetadata[name] = md
	}
	return md
}
*/

/*
Commented for now, uncomment in case you need to register a deprecated resource guarded behind a feature flag.
func newDeprecatedResourceMetadataWithFeatureFlag(name permissions.Resource, scope permissions.ResourceScope,
	replacingResourceMD permissions.ResourceMetadata, flag features.FeatureFlag) permissions.ResourceMetadata {
	md := permissions.ResourceMetadata{
		Resource:          name,
		Scope:             scope,
		ReplacingResource: &replacingResourceMD,
	}
	if flag.Enabled() {
		resourceToMetadata[name] = md
	} else {
		disabledResourceToMetadata[name] = md
	}
	return md
}
*/

func newInternalResourceMetadata(name permissions.Resource, scope permissions.ResourceScope) permissions.ResourceMetadata {
	md := permissions.ResourceMetadata{
		Resource: name,
		Scope:    scope,
	}
	internalResourceToMetadata[name] = md
	return md
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

// ListAllInternalMetadata returns a list of all resource metadata that are internal only
func ListAllInternalMetadata() []permissions.ResourceMetadata {
	metadatas := make([]permissions.ResourceMetadata, 0, len(internalResourceToMetadata))
	for _, metadata := range internalResourceToMetadata {
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

// MetadataForInternalResource returns the internal metadata for the given resource.
// If the resource is unknown, metadata for this resource with global scope is returned.
func MetadataForInternalResource(res permissions.Resource) (permissions.ResourceMetadata, bool) {
	md, found := internalResourceToMetadata[res]
	if !found {
		md.Resource = res
		md.Scope = permissions.GlobalScope
	}
	return md, found
}
