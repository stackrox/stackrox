package main

import (
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

var typeRegistry = make(map[string]string)

func init() {
	// KEEP THE FOLLOWING LIST SORTED IN LEXICOGRAPHIC ORDER (case-sensitive).
	for s, r := range map[proto.Message]permissions.ResourceHandle{
		&storage.ActiveComponent{}:                              resources.Deployment,
		&storage.AdministrationEvent{}:                          resources.Administration,
		&storage.AuthMachineToMachineConfig{}:                   resources.Access,
		&storage.AuthProvider{}:                                 resources.Access,
		&storage.Blob{}:                                         resources.Administration,
		&storage.ClusterHealthStatus{}:                          resources.Cluster,
		&storage.ClusterCVE{}:                                   resources.Cluster,
		&storage.ClusterCVEEdge{}:                               resources.Cluster,
		&storage.ComplianceConfig{}:                             resources.Compliance,
		&storage.ComplianceControlResult{}:                      resources.Compliance,
		&storage.ComplianceDomain{}:                             resources.Compliance,
		&storage.ComplianceIntegration{}:                        resources.Integration,
		&storage.ComplianceOperatorCheckResult{}:                resources.ComplianceOperator,
		&storage.ComplianceOperatorCheckResultV2{}:              resources.ComplianceOperator,
		&storage.ComplianceOperatorClusterScanConfigStatus{}:    resources.ComplianceOperator,
		&storage.ComplianceOperatorProfile{}:                    resources.ComplianceOperator,
		&storage.ComplianceOperatorProfileClusterEdge{}:         resources.ComplianceOperator,
		&storage.ComplianceOperatorProfileV2{}:                  resources.ComplianceOperator,
		&storage.ComplianceOperatorRule{}:                       resources.ComplianceOperator,
		&storage.ComplianceOperatorRuleV2{}:                     resources.ComplianceOperator,
		&storage.ComplianceOperatorScan{}:                       resources.ComplianceOperator,
		&storage.ComplianceOperatorScanConfigurationV2{}:        resources.ComplianceOperator,
		&storage.ComplianceOperatorScanV2{}:                     resources.ComplianceOperator,
		&storage.ComplianceOperatorScanSettingBinding{}:         resources.ComplianceOperator,
		&storage.ComplianceRunMetadata{}:                        resources.Compliance,
		&storage.ComplianceRunResults{}:                         resources.Compliance,
		&storage.ComplianceStrings{}:                            resources.Compliance,
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
		&storage.ReportConfiguration{}:                          resources.WorkflowAdministration,
		&storage.ReportSnapshot{}:                               resources.WorkflowAdministration,
		&storage.ResourceCollection{}:                           resources.WorkflowAdministration,
		&storage.Risk{}:                                         resources.DeploymentExtension,
		&storage.Role{}:                                         resources.Access,
		&storage.SecuredUnits{}:                                 resources.Administration,
		&storage.SensorUpgradeConfig{}:                          resources.Administration,
		&storage.ServiceIdentity{}:                              resources.Administration,
		&storage.SignatureIntegration{}:                         resources.Integration,
		&storage.SimpleAccessScope{}:                            resources.Access,
		&storage.StoredLicenseKey{}:                             resources.Access,
		&storage.SystemInfo{}:                                   resources.Administration,
		&storage.TelemetryConfiguration{}:                       resources.Administration,
		&storage.TokenMetadata{}:                                resources.Integration,
		&storage.User{}:                                         resources.Access,
		// Tests
		&storage.TestStruct{}:              resources.Namespace,
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
		typeRegistry[fmt.Sprintf("%T", s)] = string(r.GetResource())
	}
}

func storageToResource(t string) string {
	if !strings.HasPrefix(t, "*") {
		t = "*" + t
	}
	s, ok := typeRegistry[t]
	if ok {
		return s
	}
	return strings.TrimPrefix(t, "*storage.")
}

func resourceMetadataFromString(resource string) permissions.ResourceMetadata {
	for _, resourceMetadata := range resources.ListAllMetadata() {
		if string(resourceMetadata.Resource) == resource {
			return resourceMetadata
		}
	}
	for _, resourceMetadata := range resources.ListAllDisabledMetadata() {
		if string(resourceMetadata.Resource) == resource {
			return resourceMetadata
		}
	}

	for _, resourceMetadata := range resources.ListAllInternalMetadata() {
		if string(resourceMetadata.Resource) == resource {
			return resourceMetadata
		}
	}
	panic("unknown resource: " + resource + ". Please add the resource to tools/generate-helpers/pg-table-bindings/list.go.")
}

func identifierGetter(prefix string, schema *walker.Schema) string {
	if len(schema.PrimaryKeys()) == 1 {
		return schema.ID().Getter(prefix)
	}
	panic(schema.TypeName + " has multiple primary keys.")
}

func clusterGetter(prefix string, schema *walker.Schema) string {
	for _, f := range schema.Fields {
		if f.Search.FieldName == search.ClusterID.String() {
			return f.Getter(prefix)
		}
	}
	panic(schema.TypeName + " has no cluster. Is it directly scoped?")
}

func namespaceGetter(prefix string, schema *walker.Schema) string {
	for _, f := range schema.Fields {
		if f.Search.FieldName == search.Namespace.String() {
			return f.Getter(prefix)
		}
	}
	panic(schema.TypeName + " has no namespace. Is it directly and namespace scoped?")
}

func searchFieldNameInOtherSchema(f walker.Field) string {
	if searchFieldName(f) != "" {
		return searchFieldName(f)
	}
	fieldInOtherSchema, err := f.Options.Reference.FieldInOtherSchema()
	if err != nil {
		panic(err)
	}
	return searchFieldName(fieldInOtherSchema)
}

func searchFieldName(f walker.Field) string {
	return f.Search.FieldName
}

func isSacScoping(f walker.Field) bool {
	return !f.Options.PrimaryKey && (f.Search.FieldName == search.ClusterID.String() || f.Search.FieldName == search.Namespace.String())
}
