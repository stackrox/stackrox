package main

import (
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
)

var typeRegistry = make(map[string]string)

func init() {
	// KEEP THE FOLLOWING LIST SORTED IN LEXICOGRAPHIC ORDER (case-sensitive).
	for s, r := range map[proto.Message]permissions.ResourceHandle{
		&storage.ActiveComponent{}:                      resources.Deployment,
		&storage.AuthProvider{}:                         resources.Access,
		&storage.ClusterHealthStatus{}:                  resources.Cluster,
		&storage.ClusterCVE{}:                           resources.Cluster,
		&storage.ClusterCVEEdge{}:                       resources.Cluster,
		&storage.ComplianceControlResult{}:              resources.Compliance,
		&storage.ComplianceDomain{}:                     resources.Compliance,
		&storage.ComplianceStrings{}:                    resources.Compliance,
		&storage.ComplianceOperatorCheckResult{}:        resources.ComplianceOperator,
		&storage.ComplianceOperatorProfile{}:            resources.ComplianceOperator,
		&storage.ComplianceOperatorRule{}:               resources.ComplianceOperator,
		&storage.ComplianceOperatorScan{}:               resources.ComplianceOperator,
		&storage.ComplianceOperatorScanSettingBinding{}: resources.ComplianceOperator,
		&storage.ComplianceRunMetadata{}:                resources.Compliance,
		&storage.ComplianceRunResults{}:                 resources.Compliance,
		&storage.ComponentCVEEdge{}:                     resources.Image,
		// TODO: ROX-12750 Replace Config with Administration.
		&storage.Config{}:             resources.Config,
		&storage.ExternalBackup{}:     resources.Integration,
		&storage.Group{}:              resources.Access,
		&storage.ImageComponent{}:     resources.Image,
		&storage.ImageComponentEdge{}: resources.Image,
		&storage.ImageCVE{}:           resources.Image,
		&storage.ImageCVEEdge{}:       resources.Image,
		&storage.ImageIntegration{}:   resources.Integration,
		&storage.IntegrationHealth{}:  resources.Integration,
		&storage.K8SRoleBinding{}:     resources.K8sRoleBinding,
		&storage.K8SRole{}:            resources.K8sRole,
		// TODO: ROX-12750 Replace DebugLogs with Administration.
		&storage.LogImbue{}:          resources.DebugLogs,
		&storage.NamespaceMetadata{}: resources.Namespace,
		&storage.NetworkBaseline{}:   resources.DeploymentExtension,
		&storage.NetworkEntity{}:     resources.NetworkGraph,
		&storage.NetworkFlow{}:       resources.NetworkGraph,
		// TODO: ROX-12750 Replace NetworkGraphConfig with Administration.
		&storage.NetworkGraphConfig{}:                           resources.NetworkGraphConfig,
		&storage.NetworkPolicyApplicationUndoDeploymentRecord{}: resources.NetworkPolicy,
		&storage.NetworkPolicyApplicationUndoRecord{}:           resources.NetworkPolicy,
		&storage.NodeComponent{}:                                resources.Node,
		&storage.NodeComponentCVEEdge{}:                         resources.Node,
		&storage.NodeComponentEdge{}:                            resources.Node,
		&storage.NodeCVE{}:                                      resources.Node,
		&storage.Notifier{}:                                     resources.Integration,
		&storage.PermissionSet{}:                                resources.Role,
		&storage.Pod{}:                                          resources.Deployment,
		// TODO: ROX-13888 Replace Policy with WorkflowAdministration.
		&storage.Policy{}:                 resources.Policy,
		&storage.PolicyCategory{}:         resources.Policy,
		&storage.ProcessBaselineResults{}: resources.DeploymentExtension,
		&storage.ProcessBaseline{}:        resources.DeploymentExtension,
		&storage.ProcessIndicator{}:       resources.DeploymentExtension,
		&storage.ProcessListeningOnPortStorage{}:                resources.ProcessWhitelist,
		&storage.ResourceCollection{}:     resources.WorkflowAdministration,
		// TODO: ROX-13888 Replace VulnerabilityReports with WorkflowAdministration.
		&storage.ReportConfiguration{}: resources.VulnerabilityReports,
		&storage.Risk{}:                resources.DeploymentExtension,
		&storage.Role{}:                resources.Role,
		// TODO: ROX-12750 Replace SensorUpgradeConfig with Administration.
		&storage.SensorUpgradeConfig{}: resources.SensorUpgradeConfig,
		// TODO: ROX-12750 Replace ServiceIdentity with Administration.
		&storage.ServiceIdentity{}:      resources.ServiceIdentity,
		&storage.SignatureIntegration{}: resources.Integration,
		&storage.SimpleAccessScope{}:    resources.Role,
		&storage.StoredLicenseKey{}:     resources.Access,
		// TODO: ROX-12750 Replace DebugLogs with Administration.
		&storage.TelemetryConfiguration{}: resources.DebugLogs,
		&storage.TokenMetadata{}:          resources.Integration,
		&storage.User{}:                   resources.Access,
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
