package pgsearch

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
)

// ResourceType of the resource, determined according to resource metadata, schema and join type.
type ResourceType int

const (
	unknown ResourceType = iota
	joinTable
	permissionChecker
	globallyScoped
	directlyScoped
	indirectlyScoped
)

var typeRegistry = make(map[string]permissions.Resource)

func init() {
	// KEEP THE FOLLOWING LIST SORTED IN LEXICOGRAPHIC ORDER (case-sensitive).
	for s, r := range map[proto.Message]permissions.ResourceHandle{
		&storage.ActiveComponent{}:                              resources.Deployment,
		&storage.AuthProvider{}:                                 resources.Access,
		&storage.Blob{}:                                         resources.Administration,
		&storage.ClusterHealthStatus{}:                          resources.Cluster,
		&storage.ClusterCVE{}:                                   resources.Cluster,
		&storage.ClusterCVEEdge{}:                               resources.Cluster,
		&storage.ComplianceControlResult{}:                      resources.Compliance,
		&storage.ComplianceDomain{}:                             resources.Compliance,
		&storage.ComplianceStrings{}:                            resources.Compliance,
		&storage.ComplianceConfig{}:                             resources.Compliance,
		&storage.ComplianceOperatorCheckResult{}:                resources.ComplianceOperator,
		&storage.ComplianceOperatorProfile{}:                    resources.ComplianceOperator,
		&storage.ComplianceOperatorRule{}:                       resources.ComplianceOperator,
		&storage.ComplianceOperatorScan{}:                       resources.ComplianceOperator,
		&storage.ComplianceOperatorScanSettingBinding{}:         resources.ComplianceOperator,
		&storage.ComplianceRunMetadata{}:                        resources.Compliance,
		&storage.ComplianceRunResults{}:                         resources.Compliance,
		&storage.ComponentCVEEdge{}:                             resources.Image,
		&storage.Config{}:                                       resources.Administration,
		&storage.DeclarativeConfigHealth{}:                      resources.Integration,
		&storage.DelegatedRegistryConfig{}:                      resources.Administration,
		&storage.ExternalBackup{}:                               resources.Integration,
		&storage.Group{}:                                        resources.Access,
		&storage.Hash{}:                                         resources.Hash,
		&storage.ImageComponent{}:                               resources.Image,
		&storage.ImageComponentEdge{}:                           resources.Image,
		&storage.ImageCVE{}:                                     resources.Image,
		&storage.ImageCVEEdge{}:                                 resources.Image,
		&storage.ImageIntegration{}:                             resources.Integration,
		&storage.IntegrationHealth{}:                            resources.Integration,
		&storage.K8SRoleBinding{}:                               resources.K8sRoleBinding,
		&storage.K8SRole{}:                                      resources.K8sRole,
		&storage.LogImbue{}:                                     resources.Administration,
		&storage.NamespaceMetadata{}:                            resources.Namespace,
		&storage.NetworkBaseline{}:                              resources.DeploymentExtension,
		&storage.NetworkEntity{}:                                resources.NetworkGraph,
		&storage.NetworkFlow{}:                                  resources.NetworkGraph,
		&storage.NetworkGraphConfig{}:                           resources.Administration,
		&storage.NetworkPolicyApplicationUndoDeploymentRecord{}: resources.NetworkPolicy,
		&storage.NetworkPolicyApplicationUndoRecord{}:           resources.NetworkPolicy,
		&storage.NodeComponent{}:                                resources.Node,
		&storage.NodeComponentCVEEdge{}:                         resources.Node,
		&storage.NodeComponentEdge{}:                            resources.Node,
		&storage.NodeCVE{}:                                      resources.Node,
		&storage.NotificationSchedule{}:                         resources.Notifications,
		&storage.Notifier{}:                                     resources.Integration,
		&storage.PermissionSet{}:                                resources.Access,
		&storage.Pod{}:                                          resources.Deployment,
		&storage.Policy{}:                                       resources.WorkflowAdministration,
		&storage.PolicyCategory{}:                               resources.WorkflowAdministration,
		&storage.PolicyCategoryEdge{}:                           resources.WorkflowAdministration,
		&storage.ProcessBaselineResults{}:                       resources.DeploymentExtension,
		&storage.ProcessBaseline{}:                              resources.DeploymentExtension,
		&storage.ProcessIndicator{}:                             resources.DeploymentExtension,
		&storage.ProcessListeningOnPortStorage{}:                resources.DeploymentExtension,
		&storage.ResourceCollection{}:                           resources.WorkflowAdministration,
		&storage.ReportConfiguration{}:                          resources.WorkflowAdministration,
		&storage.Risk{}:                                         resources.DeploymentExtension,
		&storage.Role{}:                                         resources.Access,
		&storage.SensorUpgradeConfig{}:                          resources.Administration,
		&storage.ServiceIdentity{}:                              resources.Administration,
		&storage.SignatureIntegration{}:                         resources.Integration,
		&storage.SimpleAccessScope{}:                            resources.Access,
		&storage.StoredLicenseKey{}:                             resources.Access,
		&storage.TelemetryConfiguration{}:                       resources.Administration,
		&storage.TokenMetadata{}:                                resources.Integration,
		&storage.User{}:                                         resources.Access,
		// Tests
		&storage.TestMultiKeyStruct{}:      resources.Namespace,
		&storage.TestSingleKeyStruct{}:     resources.Namespace,
		&storage.TestSingleUUIDKeyStruct{}: resources.Namespace,
		&storage.TestGrandparent{}:         resources.Namespace,
		&storage.TestParent1{}:             resources.Namespace,
		&storage.TestChild1{}:              resources.Namespace,
		&storage.TestGrandChild1{}:         resources.Namespace,
		&storage.TestGGrandChild1{}:        resources.Namespace,
		&storage.TestG2GrandChild1{}:       resources.Namespace,
		&storage.TestG3GrandChild1{}:       resources.Namespace,
		&storage.TestParent2{}:             resources.Namespace,
		&storage.TestChild2{}:              resources.Namespace,
		&storage.TestParent3{}:             resources.Namespace,
		&storage.TestParent4{}:             resources.Namespace,
		&storage.TestChild1P4{}:            resources.Namespace,
		&storage.TestShortCircuit{}:        resources.Namespace,
	} {
		typeRegistry[fmt.Sprintf("%T", s)] = r.GetResource()
	}
}

// TODO(janisz): add cache
func resourceMetadata[T any](t T) (permissions.ResourceMetadata, error) {
	resource, ok := typeRegistry[fmt.Sprintf("%T", t)]
	if !ok {
		return permissions.ResourceMetadata{}, fmt.Errorf("unregistered type %T", t)
	}
	for _, resourceMetadata := range resources.ListAllMetadata() {
		if resourceMetadata.Resource == resource {
			return resourceMetadata, nil
		}
	}
	for _, resourceMetadata := range resources.ListAllDisabledMetadata() {
		if resourceMetadata.Resource == resource {
			return resourceMetadata, nil
		}
	}

	for _, resourceMetadata := range resources.ListAllInternalMetadata() {
		if resourceMetadata.Resource == resource {
			return resourceMetadata, nil
		}
	}
	return permissions.ResourceMetadata{}, fmt.Errorf("unknown resource %s", resource)
}

type unmarshaler[T any] interface {
	proto.Unmarshaler
	*T
}

// GetReadWriteSACQuery returns SAC filter for resource or error is permission is denied.
func GetReadWriteSACQuery[T any, PT unmarshaler[T]](ctx context.Context, t PT) (*v1.Query, error) {
	metadata, err := resourceMetadata(t)
	if err != nil {
		return nil, err
	}
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_WRITE_ACCESS).Resource(metadata)
	switch metadata.GetScope() {
	case permissions.GlobalScope:
		if !scopeChecker.IsAllowed() {
			return nil, sac.ErrResourceAccessDenied
		}
		return &v1.Query{}, nil
	case permissions.ClusterScope:
		scopeTree, err := scopeChecker.EffectiveAccessScope(permissions.Modify(metadata))
		if err != nil {
			return nil, err
		}
		return sac.BuildNonVerboseClusterLevelSACQueryFilter(scopeTree)
	case permissions.NamespaceScope:
		scopeTree, err := scopeChecker.EffectiveAccessScope(permissions.Modify(metadata))
		if err != nil {
			return nil, err
		}
		return sac.BuildNonVerboseClusterNamespaceLevelSACQueryFilter(scopeTree)
	}
	return nil, fmt.Errorf("could not prepare SAC Query for %s", metadata)
}
