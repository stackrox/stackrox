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
	// Access is the new resource grouping all access related resources.
	Access = newResourceMetadata("Access", permissions.GlobalScope)
	// Administration is the new resource grouping all administration-like resources.
	Administration = newResourceMetadata("Administration", permissions.GlobalScope)
	Alert          = newResourceMetadata("Alert", permissions.NamespaceScope)
	CVE            = newResourceMetadata("CVE", permissions.NamespaceScope)
	Cluster        = newResourceMetadata("Cluster", permissions.ClusterScope)
	Compliance     = newResourceMetadata("Compliance", permissions.GlobalScope)
	Deployment     = newResourceMetadata("Deployment", permissions.NamespaceScope)
	// DeploymentExtensions is the new resource grouping all deployment extending resources.
	DeploymentExtensions = newResourceMetadata("DeploymentExtensions", permissions.NamespaceScope)
	Detection            = newResourceMetadata("Detection", permissions.GlobalScope)
	Image                = newResourceMetadata("Image", permissions.NamespaceScope)
	ImageComponent       = newResourceMetadata("ImageComponent", permissions.NamespaceScope)
	// Integrations is the new  resource grouping all integration resources.
	Integrations                     = newResourceMetadata("Integrations", permissions.GlobalScope)
	Indicator                        = newResourceMetadata("Indicator", permissions.NamespaceScope)
	K8sRole                          = newResourceMetadata("K8sRole", permissions.NamespaceScope)
	K8sRoleBinding                   = newResourceMetadata("K8sRoleBinding", permissions.NamespaceScope)
	K8sSubject                       = newResourceMetadata("K8sSubject", permissions.NamespaceScope)
	Namespace                        = newResourceMetadata("Namespace", permissions.NamespaceScope)
	NetworkGraph                     = newResourceMetadata("NetworkGraph", permissions.NamespaceScope)
	NetworkPolicy                    = newResourceMetadata("NetworkPolicy", permissions.NamespaceScope)
	Node                             = newResourceMetadata("Node", permissions.ClusterScope)
	Policy                           = newResourceMetadata("Policy", permissions.GlobalScope)
	Secret                           = newResourceMetadata("Secret", permissions.NamespaceScope)
	ServiceAccount                   = newResourceMetadata("ServiceAccount", permissions.NamespaceScope)
	VulnerabilityManagementApprovals = newResourceMetadata("VulnerabilityManagementApprovals", permissions.GlobalScope)
	VulnerabilityManagementRequests  = newResourceMetadata("VulnerabilityManagementRequests", permissions.GlobalScope)
	VulnerabilityReports             = newResourceMetadata("VulnerabilityReports", permissions.GlobalScope)
	WatchedImage                     = newResourceMetadata("WatchedImage", permissions.GlobalScope)

	// Deprecated resources.

	// Deprecated: AllComments is deprecated, use Administration.
	AllComments = newResourceMetadata("AllComments", permissions.GlobalScope)
	// Deprecated: APIToken is deprecated, use Integrations.
	APIToken = newResourceMetadata("APIToken", permissions.GlobalScope)
	// Deprecated: AuthPlugin is deprecated, use Access.
	AuthPlugin = newResourceMetadata("AuthPlugin", permissions.GlobalScope)
	// Deprecated: AuthProvider is deprecated, use Access.
	AuthProvider = newResourceMetadata("AuthProvider", permissions.GlobalScope)
	// Deprecated: BackupPlugins is deprecated, use Integrations.
	BackupPlugins = newResourceMetadata("BackupPlugins", permissions.GlobalScope)
	// Deprecated: ComplianceRuns is deprecated, use Compliance.
	ComplianceRuns = newResourceMetadata("ComplianceRuns", permissions.ClusterScope)
	// Deprecated: ComplianceRunSchedule is deprecated, use Administration.
	ComplianceRunSchedule = newResourceMetadata("ComplianceRunSchedule", permissions.GlobalScope)
	// Deprecated: Config is deprecated, use Administration.
	Config = newResourceMetadata("Config", permissions.GlobalScope)
	// Deprecated: DebugLogs is deprecated, use Administration.
	DebugLogs = newResourceMetadata("DebugLogs", permissions.GlobalScope)
	// Deprecated: Group is deprecated, use Access.
	Group = newResourceMetadata("Group", permissions.GlobalScope)
	// Deprecated: ImageIntegration is deprecated, use Integrations.
	ImageIntegration = newResourceMetadata("ImageIntegration", permissions.GlobalScope)
	// Deprecated: Licenses is deprecated, use Access.
	Licenses = newResourceMetadata("Licenses", permissions.GlobalScope)
	// Deprecated: NetworkBaseline is deprecated, use DeploymentExtensions.
	NetworkBaseline = newResourceMetadata("NetworkBaseline", permissions.NamespaceScope)
	// Deprecated: NetworkGraphConfig is deprecated, use Administration.
	NetworkGraphConfig = newResourceMetadata("NetworkGraphConfig", permissions.GlobalScope)
	// Deprecated: Notifier is deprecated, use Integrations.
	Notifier = newResourceMetadata("Notifier", permissions.GlobalScope)
	// Deprecated: ProbeUpload is deprecated, use Administration.
	ProbeUpload = newResourceMetadata("ProbeUpload", permissions.GlobalScope)
	// Deprecated: ProcessWhitelist is deprecated, use DeploymentExtensions.
	ProcessWhitelist = newResourceMetadata("ProcessWhitelist", permissions.NamespaceScope)
	// Deprecated: Risk is deprecated, use DeploymentExtensions.
	Risk = newResourceMetadata("Risk", permissions.NamespaceScope)
	// Deprecated: Role is deprecated, use Access.
	Role = newResourceMetadata("Role", permissions.GlobalScope)
	// Deprecated: ScannerBundle is deprecated, use Administration.
	ScannerBundle = newResourceMetadata("ScannerBundle", permissions.GlobalScope)
	// Deprecated: ScannerDefinitions is deprecated, use Administration.
	ScannerDefinitions = newResourceMetadata("ScannerDefinitions", permissions.GlobalScope)
	// Deprecated: SensorUpgradeConfig is deprecated, use Administration.
	SensorUpgradeConfig = newResourceMetadata("SensorUpgradeConfig", permissions.GlobalScope)
	// Deprecated: ServiceIdentity is deprecated, use Administration.
	ServiceIdentity = newResourceMetadata("ServiceIdentity", permissions.GlobalScope)
	// Deprecated: SignatureIntegration is deprecated, use Integrations.
	SignatureIntegration = newResourceMetadataWithFeatureFlag("SignatureIntegration", permissions.GlobalScope, features.ImageSignatureVerification)
	// Deprecated: User is deprecated, use Access.
	User = newResourceMetadata("User", permissions.GlobalScope)

	// Internal Resources.
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

/*
Resource replacements -> how would this be best?
permission.ResourceMetadata is the sole single struct used (and we should keep it this way).
*/
