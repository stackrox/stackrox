package mapping

import (
	"fmt"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func getActiveComponentPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "activecomponent"
	}
	return "active_component"
}

func getAlertPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "alert"
	}
	return "list_alert"
}

func getClusterPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "cluster"
	}
	return "cluster"
}

func getClusterCVEPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "clustercve"
	}
	return "c_v_e"
}

func getClusterCVEEdgePrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "clustercveedge"
	}
	return "cluster_c_v_e_edge"
}

func getComponentVulnEdgePrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "componentcveedge"
	}
	return "component_c_v_e_edge"
}

func getDeploymentPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "deployment"
	}
	return "deployment"
}

func getImagePrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "image"
	}
	return "image"
}

func getImageComponentPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "imagecomponent"
	}
	return "image_component"
}

func getImageComponentEdgePrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "imagecomponentedge"
	}
	return "imagecomponentedge"
}

func getImageCVEPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "imagecve"
	}
	return "c_v_e"
}

func getImageCVEEdgePrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "imagecveedge"
	}
	return "image_c_v_e_edge"
}

func getImageIntegrationPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "imageintegration"
	}
	return "image_integration"
}

func getNamespacePrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "namespacemetadata"
	}
	return "namespace_metadata"
}

func getNodePrefix() string {
	return "node"
}

func getNodeComponentPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "nodecomponent"
	}
	return "image_component"
}

func getNodeComponentEdgePrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "nodecomponentedge"
	}
	return "imagecomponentedge"
}

func getNodeComponentVulnEdgePrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "nodecomponentcveedge"
	}
	return "component_c_v_e_edge"
}

func getNodeCVEPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "nodecve"
	}
	return "c_v_e"
}

func getProcessIndicatorPrefix() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return "processindicator"
	}
	return "process_indicator"
}

func getImageCVESearchCategory() v1.SearchCategory {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return v1.SearchCategory_IMAGE_VULNERABILITIES
	}
	return v1.SearchCategory_VULNERABILITIES
}

func getNodeComponentSearchCategory() v1.SearchCategory {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return v1.SearchCategory_NODE_COMPONENTS
	}
	return v1.SearchCategory_IMAGE_COMPONENTS
}

func getNodeComponentCVEEdgeSearchCategory() v1.SearchCategory {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return v1.SearchCategory_NODE_COMPONENT_CVE_EDGE
	}
	return v1.SearchCategory_COMPONENT_VULN_EDGE
}

func getNodeCVESearchCategory() v1.SearchCategory {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return v1.SearchCategory_NODE_VULNERABILITIES
	}
	return v1.SearchCategory_VULNERABILITIES
}

var (
	// Field Values - ActiveComponent
	activeComponentContainerNameField = &search.Field{
		FieldPath: getActiveComponentPrefix() + ".active_contexts_slice.container_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_ACTIVE_COMPONENT,
		Analyzer:  "",
	}
	activeComponentDeploymentIDField = &search.Field{
		FieldPath: getActiveComponentPrefix() + ".deployment_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_ACTIVE_COMPONENT,
		Analyzer:  "",
	}
	activeComponentIDField = &search.Field{
		FieldPath: getActiveComponentPrefix() + ".component_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_ACTIVE_COMPONENT,
		Analyzer:  "",
	}
	activeComponentImageIDField = &search.Field{
		FieldPath: getActiveComponentPrefix() + ".active_contexts_slice.image_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_ACTIVE_COMPONENT,
		Analyzer:  "",
	}
	// Field Values - Alert
	alertCategoryField = &search.Field{
		FieldPath: getAlertPrefix() + ".policy.categories",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertClusterIDLegacyField = &search.Field{
		FieldPath: getAlertPrefix() + ".common_entity_info.cluster_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertClusterIDPostgresField = &search.Field{
		FieldPath: getAlertPrefix() + ".cluster_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertClusterNameLegacyField = &search.Field{
		FieldPath: getAlertPrefix() + ".common_entity_info.cluster_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertClusterNamePostgresField = &search.Field{
		FieldPath: getAlertPrefix() + ".cluster_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertDeploymentIDField = &search.Field{
		FieldPath: getAlertPrefix() + ".Entity.deployment.id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertDeploymentNameField = &search.Field{
		FieldPath: getAlertPrefix() + ".Entity.deployment.name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertEnforcementLegacyField = &search.Field{
		FieldPath: getAlertPrefix() + ".enforcement_action",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertEnforcementPostgresField = &search.Field{
		FieldPath: getAlertPrefix() + ".enforcement.action",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertInactiveField = &search.Field{
		FieldPath: getAlertPrefix() + ".Entity.deployment.inactive",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertLifecycleStageLegacyField = &search.Field{
		FieldPath: getAlertPrefix() + ".lifecycle_stage",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertLifecycleStagePostgresField = &search.Field{
		FieldPath: getAlertPrefix() + ".lifecycle_stage",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertNamespaceIDLegacyField = &search.Field{
		FieldPath: getAlertPrefix() + ".common_entity_info.namespace_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertNamespaceIDPostgresField = &search.Field{
		FieldPath: getAlertPrefix() + ".namespace_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertNamespaceNameLegacyField = &search.Field{
		FieldPath: getAlertPrefix() + ".common_entity_info.namespace",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertNamespaceNamePostgresField = &search.Field{
		FieldPath: getAlertPrefix() + ".namespace",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertPolicyIDField = &search.Field{
		FieldPath: getAlertPrefix() + ".policy.id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertPolicyNameField = &search.Field{
		FieldPath: getAlertPrefix() + ".policy.name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertPolicySeverityField = &search.Field{
		FieldPath: getAlertPrefix() + ".policy.severity",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertResourceNameField = &search.Field{
		FieldPath: getAlertPrefix() + ".Entity.resource.name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertResourceTypeLegacyField = &search.Field{
		FieldPath: getAlertPrefix() + ".common_entity_info.resource_type",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertResourceTypePostgresField = &search.Field{
		FieldPath: getAlertPrefix() + ".Entity.resource.resource_type",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertSortPolicyNameLegacyField = &search.Field{
		FieldPath: getAlertPrefix() + ".policy.developer_internal_fields.SORT_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "keyword",
	}
	alertSortPolicyNamePostgresField = &search.Field{
		FieldPath: getAlertPrefix() + ".policy.SORT_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "keyword",
	}
	alertStateField = &search.Field{
		FieldPath: getAlertPrefix() + ".state",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertViolationTimeField = &search.Field{
		FieldPath: getAlertPrefix() + ".time.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	// Field Values - Cluster
	clusterAdmissionControlStatusField = &search.Field{
		FieldPath: getClusterPrefix() + ".health_status.admission_control_health_status",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	clusterClusterStatusField = &search.Field{
		FieldPath: getClusterPrefix() + ".health_status.overall_health_status",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	clusterCollectorStatusField = &search.Field{
		FieldPath: getClusterPrefix() + ".health_status.collector_health_status",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	clusterIDField = &search.Field{
		FieldPath: getClusterPrefix() + ".id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	clusterLabelsField = &search.Field{
		FieldPath: getClusterPrefix() + ".labels",
		Type:      v1.SearchDataType_SEARCH_MAP,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	clusterLastContactField = &search.Field{
		FieldPath: getClusterPrefix() + ".health_status.last_contact.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	clusterNameField = &search.Field{
		FieldPath: getClusterPrefix() + ".name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	clusterScannerStatusField = &search.Field{
		FieldPath: getClusterPrefix() + ".health_status.scanner_health_status",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	clusterSensorStatusField = &search.Field{
		FieldPath: getClusterPrefix() + ".health_status.sensor_health_status",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	// Field Values - ClusterCVE
	clusterCVECVEField = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".cve_base_info.cve",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	clusterCVECVSSField = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".cvss",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	clusterCVECreatedTimeField = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".cve_base_info.created_at.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	clusterCVEIDField = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	clusterCVEImpactScore = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".impact_score",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	clusterCVEPublishedOnField = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".cve_base_info.published_on.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	ClusterCVESeverityField = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".severity",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	clusterCVESnoozedField = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".snoozed",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	clusterCVESnoozeExpiryField = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".snooze_expiry.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	clusterCVETypeField = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".type",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	// Field Values - ClusterCVEEdge
	clusterCVEEdgeFixableField = &search.Field{
		FieldPath: getClusterCVEEdgePrefix() + ".is_fixable",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_CLUSTER_VULN_EDGE,
		Analyzer:  "",
	}
	clusterCVEEdgeFixedByField = &search.Field{
		FieldPath: getClusterCVEEdgePrefix() + ".HasFixedBy.fixed_by",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_CLUSTER_VULN_EDGE,
		Analyzer:  "",
	}
	// Field Values - ComplianceControl
	complianceControlGroupIDField = &search.Field{
		FieldPath: "control.group_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_COMPLIANCE_CONTROL,
		Analyzer:  "",
	}
	complianceControlIDField = &search.Field{
		FieldPath: "control.id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_COMPLIANCE_CONTROL,
		Analyzer:  "",
	}
	complianceControlNameField = &search.Field{
		FieldPath: "control.name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_COMPLIANCE_CONTROL,
		Analyzer:  "",
	}
	complianceControlStandardIDField = &search.Field{
		FieldPath: "control.standard_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_COMPLIANCE_CONTROL,
		Analyzer:  "",
	}
	// Field Values - ComplianceStandard
	complianceStandardIDField = &search.Field{
		FieldPath: "standard.metadata.id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_COMPLIANCE_STANDARD,
		Analyzer:  "",
	}
	complianceStandardNameField = &search.Field{
		FieldPath: "standard.metadata.name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_COMPLIANCE_STANDARD,
		Analyzer:  "",
	}
	// Field Values - legacy CVE
	cveLegacyObjCreatedTimeField = &search.Field{
		FieldPath: "c_v_e.created_at.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_VULNERABILITIES,
		Analyzer:  "",
	}
	cveLegacyObjCVSSField = &search.Field{
		FieldPath: "c_v_e.cvss",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_VULNERABILITIES,
		Analyzer:  "",
	}
	cveLegacyObjIDField = &search.Field{
		FieldPath: "c_v_e.id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_VULNERABILITIES,
		Analyzer:  "",
	}
	cveLegacyObjImpactScoreField = &search.Field{
		FieldPath: "c_v_e.impact_score",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_VULNERABILITIES,
		Analyzer:  "",
	}
	cveLegacyObjPublishedOnField = &search.Field{
		FieldPath: "c_v_e.published_on.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_VULNERABILITIES,
		Analyzer:  "",
	}
	cveLegacyObjSeverityField = &search.Field{
		FieldPath: "c_v_e.severity",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_VULNERABILITIES,
		Analyzer:  "",
	}
	cveLegacyObjSnoozedField = &search.Field{
		FieldPath: "c_v_e.suppressed",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_VULNERABILITIES,
		Analyzer:  "",
	}
	cveLegacyObjSnoozeExpiryField = &search.Field{
		FieldPath: "c_v_e.suppress_expiry.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_VULNERABILITIES,
		Analyzer:  "",
	}
	cveLegacyObjTypeField = &search.Field{
		FieldPath: "c_v_e.types",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_VULNERABILITIES,
		Analyzer:  "",
	}

	cveLegacyObjCVEField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".cve_base_info.cve",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	cveLegacyObjOperatingSystemField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".operating_system",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}

	// Field Values - Deployment
	deploymentAddCapabilitiesField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.security_context.add_capabilities",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentAnnotationsField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".annotations",
		Type:      v1.SearchDataType_SEARCH_MAP,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentClusterIDField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".cluster_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentClusterLabelField = &search.Field{
		FieldPath: getClusterPrefix() + ".labels",
		Type:      v1.SearchDataType_SEARCH_MAP,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	deploymentClusterNameField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".cluster_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentCPUCoresLimitField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.resources.cpu_cores_limit",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentCPUCoresRequestField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.resources.cpu_cores_request",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentCreatedField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".created.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentDropCapabilitiesField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.security_context.drop_capabilities",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentEnvKeyField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.config.env.key",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentEnvValueField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.config.env.value",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentEnvVarSourceField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.config.env.env_var_source",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentExposedNodePortField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.node_port",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentExposingServiceField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.service_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentExposingServicePortField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.service_port",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentExposureLevelField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.level",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentExternalHostnameField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.external_hostnames",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentExternalIPField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.external_ips",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentIDField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentImageIDField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.image.id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentImageNameField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.image.name.full_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "standard",
	}
	deploymentImagePullSecretsField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".image_pull_secrets",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentImageRegistryField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.image.name.registry",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentImageRemoteField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.image.name.remote",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentImageTagField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.image.name.tag",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentLabelsField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".labels",
		Type:      v1.SearchDataType_SEARCH_MAP,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentMaxExposureField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".ports.exposure",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentMemoryLimitField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.resources.memory_mb_limit",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentMemoryRequestField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.resources.memory_mb_request",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentNameField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentNamespaceField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".namespace",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentNamespaceIDField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".namespace_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentOrchestratorComponentField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".orchestrator_component",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentPodLabelField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".pod_labels",
		Type:      v1.SearchDataType_SEARCH_MAP,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentPortField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".ports.container_port",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentPortProtocolField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".ports.protocol",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentPriorityField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".priority",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentPrivilegedField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.security_context.privileged",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentReadOnlyRootFilesystemField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.security_context.read_only_root_filesystem",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentRiskScoreField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".risk_score",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentSecretNameField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.secrets.name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentSecretPathField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.secrets.path",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentServiceAccountNameField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".service_account",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentServiceAccountPermissionLevelField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".service_account_permission_level",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentTypeField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".type",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentVolumeDestinationField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.volumes.destination",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentVolumeNameField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.volumes.name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentVolumeReadOnlyField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.volumes.read_only",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentVolumeSourceField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.volumes.source",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentVolumeTypeField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.volumes.type",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	// Field Values - Image
	imageObjCommandField = &search.Field{
		FieldPath: getImagePrefix() + ".metadata.v1.command",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjComponentCountField = &search.Field{
		FieldPath: getImagePrefix() + ".SetComponents.components",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjComponentNameField = &search.Field{
		FieldPath: getImagePrefix() + ".scan.components.name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjComponentRiskScoreField = &search.Field{
		FieldPath: getImagePrefix() + ".scan.components.risk_score",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjComponentVersionField = &search.Field{
		FieldPath: getImagePrefix() + ".scan.components.version",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjCreatedTimeField = &search.Field{
		FieldPath: getImagePrefix() + ".metadata.v1.created.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjCVEField = &search.Field{
		FieldPath: getImagePrefix() + ".scan.components.vulns.cve",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjCVECountField = &search.Field{
		FieldPath: getImagePrefix() + ".SetCves.cves",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjCVEPublishedOnField = &search.Field{
		FieldPath: getImagePrefix() + ".scan.components.vulns.published_on.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjCVEStateField = &search.Field{
		FieldPath: getImagePrefix() + ".scan.components.vulns.state",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjCVESuppressedField = &search.Field{
		FieldPath: getImagePrefix() + ".scan.components.vulns.suppressed",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjCVSSField = &search.Field{
		FieldPath: getImagePrefix() + ".scan.components.vulns.cvss",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjDockerfileInstructionField = &search.Field{
		FieldPath: getImagePrefix() + ".metadata.v1.layers.instruction",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjDockerfileInstructionValueField = &search.Field{
		FieldPath: getImagePrefix() + ".metadata.v1.layers.value",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjEntrypointField = &search.Field{
		FieldPath: getImagePrefix() + ".metadata.v1.entrypoint",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjFixableCVEsField = &search.Field{
		FieldPath: getImagePrefix() + ".SetFixable.fixable_cves",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjFixedByField = &search.Field{
		FieldPath: getImagePrefix() + ".scan.components.vulns.SetFixedBy.fixed_by",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjLabelField = &search.Field{
		FieldPath: getImagePrefix() + ".metadata.v1.labels",
		Type:      v1.SearchDataType_SEARCH_MAP,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjLastUpdatedField = &search.Field{
		FieldPath: getImagePrefix() + ".last_updated.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjNameField = &search.Field{
		FieldPath: getImagePrefix() + ".name.full_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "standard",
	}
	imageObjOperatingSystemField = &search.Field{
		FieldPath: getImagePrefix() + ".scan.operating_system",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjPriorityField = &search.Field{
		FieldPath: getImagePrefix() + ".priority",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjRegistryField = &search.Field{
		FieldPath: getImagePrefix() + ".name.registry",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjRemoteField = &search.Field{
		FieldPath: getImagePrefix() + ".name.remote",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjRiskScoreField = &search.Field{
		FieldPath: getImagePrefix() + ".risk_score",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjScanTimeField = &search.Field{
		FieldPath: getImagePrefix() + ".scan.scan_time.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjIDField = &search.Field{
		FieldPath: getImagePrefix() + ".id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjSignatureFetchTimeField = &search.Field{
		FieldPath: getImagePrefix() + ".signature.fetched.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjTagField = &search.Field{
		FieldPath: getImagePrefix() + ".name.tag",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjTopCVSSField = &search.Field{
		FieldPath: getImagePrefix() + ".SetTopCvss.top_cvss",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjUserField = &search.Field{
		FieldPath: getImagePrefix() + ".metadata.v1.user",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	imageObjVolumesField = &search.Field{
		FieldPath: getImagePrefix() + ".metadata.v1.volumes",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGES,
		Analyzer:  "",
	}
	// Field Values - ImageComponent
	imageComponentObjIDField = &search.Field{
		FieldPath: getImageComponentPrefix() + ".id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_IMAGE_COMPONENTS,
		Analyzer:  "",
	}
	imageComponentObjNameField = &search.Field{
		FieldPath: getImageComponentPrefix() + ".name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGE_COMPONENTS,
		Analyzer:  "",
	}
	imageComponentObjOperatingSystemField = &search.Field{
		FieldPath: getImageComponentPrefix() + ".operating_system",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGE_COMPONENTS,
		Analyzer:  "",
	}
	imageComponentObjPriorityField = &search.Field{
		FieldPath: getImageComponentPrefix() + ".priority",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_IMAGE_COMPONENTS,
		Analyzer:  "",
	}
	imageComponentObjRiskScoreField = &search.Field{
		FieldPath: getImageComponentPrefix() + ".risk_score",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_IMAGE_COMPONENTS,
		Analyzer:  "",
	}
	imageComponentObjSourceField = &search.Field{
		FieldPath: getImageComponentPrefix() + ".source",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGE_COMPONENTS,
		Analyzer:  "",
	}
	imageComponentObjTopCVSSField = &search.Field{
		FieldPath: getImageComponentPrefix() + ".SetTopCvss.top_cvss",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGE_COMPONENTS,
		Analyzer:  "",
	}
	imageComponentObjVersionField = &search.Field{
		FieldPath: getImageComponentPrefix() + ".version",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGE_COMPONENTS,
		Analyzer:  "",
	}
	// Field Values - ImageComponentCVEEdge
	imageComponentCVEEdgeFixableField = &search.Field{
		FieldPath: getComponentVulnEdgePrefix() + ".is_fixable",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_COMPONENT_VULN_EDGE,
		Analyzer:  "",
	}
	imageComponentCVEEdgeFixedByField = &search.Field{
		FieldPath: getComponentVulnEdgePrefix() + ".HasFixedBy.fixed_by",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_COMPONENT_VULN_EDGE,
		Analyzer:  "",
	}
	// Field Values - ImageComponentEdge
	imageComponentEdgeLocationField = &search.Field{
		FieldPath: getImageComponentEdgePrefix() + ".location",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_IMAGE_COMPONENT_EDGE,
		Analyzer:  "",
	}
	// Field Values - ImageCVE
	imageCVECreatedTimeField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".cve_base_info.created_at.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	imageCVECVEField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".cve_base_info.cve",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	imageCVECVSSField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".cvss",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	imageCVEIDField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	imageCVEImpactScoreField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".impact_score",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    false,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	imageCVEOperatingSystemField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".operating_system",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	imageCVEPublishedOnField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".cve_base_info.published_on.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	imageCVESeverityField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".severity",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	imageCVESnoozedField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".snoozed",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     false,
		Hidden:    false,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	imageCVESnoozeExpiryField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".snooze_expiry.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    true,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	// Field Values - ImageCVEEdge
	imageCVEEdgeFirstOccurrenceField = &search.Field{
		FieldPath: getImageCVEEdgePrefix() + ".first_image_occurrence.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_IMAGE_VULN_EDGE,
		Analyzer:  "",
	}
	imageCVEEdgeStateField = &search.Field{
		FieldPath: getImageCVEEdgePrefix() + ".state",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_IMAGE_VULN_EDGE,
		Analyzer:  "",
	}
	// Field Values - ImageIntegration
	imageIntegrationObjClusterIDField = &search.Field{
		FieldPath: getImageIntegrationPrefix() + ".cluster_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_IMAGE_INTEGRATIONS,
		Analyzer:  "",
	}
	// Field Values - Namespace
	namespaceAnnotationsField = &search.Field{
		FieldPath: getNamespacePrefix() + ".annotations",
		Type:      v1.SearchDataType_SEARCH_MAP,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_NAMESPACES,
		Analyzer:  "",
	}
	namespaceClusterField = &search.Field{
		FieldPath: getNamespacePrefix() + ".cluster_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_NAMESPACES,
		Analyzer:  "",
	}
	namespaceClusterIDField = &search.Field{
		FieldPath: getNamespacePrefix() + ".cluster_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_NAMESPACES,
		Analyzer:  "",
	}
	namespaceIDField = &search.Field{
		FieldPath: getNamespacePrefix() + ".id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_NAMESPACES,
		Analyzer:  "",
	}
	namespaceLabelField = &search.Field{
		FieldPath: getNamespacePrefix() + ".labels",
		Type:      v1.SearchDataType_SEARCH_MAP,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_NAMESPACES,
		Analyzer:  "",
	}
	namespaceNameField = &search.Field{
		FieldPath: getNamespacePrefix() + ".name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_NAMESPACES,
		Analyzer:  "",
	}
	// Field Values - Node
	nodeObjAnnotationField = &search.Field{
		FieldPath: getNodePrefix() + ".annotations",
		Type:      v1.SearchDataType_SEARCH_MAP,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjClusterIDField = &search.Field{
		FieldPath: getNodePrefix() + ".cluster_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjClusterNameField = &search.Field{
		FieldPath: getNodePrefix() + ".cluster_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjComponentField = &search.Field{
		FieldPath: getNodePrefix() + ".scan.components.name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjComponentCountField = &search.Field{
		FieldPath: getNodePrefix() + ".SetComponents.components",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjComponentVersionField = &search.Field{
		FieldPath: getNodePrefix() + ".scan.components.version",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjContainerRuntimeVersionField = &search.Field{
		FieldPath: getNodePrefix() + ".container_runtime.version",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjCVEField = &search.Field{
		FieldPath: getNodePrefix() + ".scan.components.vulnerabilities.cve_base_info.cve",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjCVECountField = &search.Field{
		FieldPath: getNodePrefix() + ".SetCves.cves",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjCVECreatedTimeField = &search.Field{
		FieldPath: getNodePrefix() + ".scan.components.vulnerabilities.cve_base_info.created_at.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjCVEPublishedOnField = &search.Field{
		FieldPath: getNodePrefix() + ".scan.components.vulnerabilities.cve_base_info.published_on.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjCVESnoozedField = &search.Field{
		FieldPath: getNodePrefix() + ".scan.components.vulns.suppressed",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjCVSSField = &search.Field{
		FieldPath: getNodePrefix() + ".scan.components.vulns.cvss",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjFixableCVECountField = &search.Field{
		FieldPath: getNodePrefix() + ".SetFixable.fixable_cves",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjFixedByField = &search.Field{
		FieldPath: getNodePrefix() + ".scan.components.vulns.SetFixedBy.fixed_by",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjIDField = &search.Field{
		FieldPath: getNodePrefix() + ".id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjJoinTimeField = &search.Field{
		FieldPath: getNodePrefix() + ".joined_at.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjLabelField = &search.Field{
		FieldPath: getNodePrefix() + ".labels",
		Type:      v1.SearchDataType_SEARCH_MAP,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjLastUpdatedField = &search.Field{
		FieldPath: getNodePrefix() + ".last_updated.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjNameField = &search.Field{
		FieldPath: getNodePrefix() + ".name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjNodeRiskPriorityField = &search.Field{
		FieldPath: getNodePrefix() + ".priority",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjOperatingSystemField = &search.Field{
		FieldPath: getNodePrefix() + ".os_image",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjRiskScoreField = &search.Field{
		FieldPath: getNodePrefix() + ".risk_score",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjNodeScanTimeField = &search.Field{
		FieldPath: getNodePrefix() + ".scan.scan_time.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjTaintEffectField = &search.Field{
		FieldPath: getNodePrefix() + ".taints.taint_effect",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjTaintKeyField = &search.Field{
		FieldPath: getNodePrefix() + ".taints.key",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjTaintValueField = &search.Field{
		FieldPath: getNodePrefix() + ".taints.value",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjTopCVSSField = &search.Field{
		FieldPath: getNodePrefix() + ".SetTopCvss.top_cvss",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	nodeObjVulnerabilityStateField = &search.Field{
		FieldPath: getNodePrefix() + ".scan.components.vulns.state",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_NODES,
		Analyzer:  "",
	}
	// Field Values - NodeComponent
	nodeComponentObjIDField = &search.Field{
		FieldPath: getNodeComponentPrefix() + ".id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  getNodeComponentSearchCategory(),
		Analyzer:  "",
	}
	nodeComponentObjNameField = &search.Field{
		FieldPath: getNodeComponentPrefix() + ".name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  getNodeComponentSearchCategory(),
		Analyzer:  "",
	}
	nodeComponentObjOperatingSystemField = &search.Field{
		FieldPath: getNodeComponentPrefix() + ".operating_system",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  getNodeComponentSearchCategory(),
		Analyzer:  "",
	}
	nodeComponentObjPriorityField = &search.Field{
		FieldPath: getNodeComponentPrefix() + ".priority",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    true,
		Category:  getNodeComponentSearchCategory(),
		Analyzer:  "",
	}
	nodeComponentObjRiskScoreField = &search.Field{
		FieldPath: getNodeComponentPrefix() + ".risk_score",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    true,
		Category:  getNodeComponentSearchCategory(),
		Analyzer:  "",
	}
	nodeComponentObjSourceField = &search.Field{
		FieldPath: getNodeComponentPrefix() + ".source",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  getNodeComponentSearchCategory(),
		Analyzer:  "",
	}
	nodeComponentObjTopCVSSField = &search.Field{
		FieldPath: getNodeComponentPrefix() + ".SetTopCvss.top_cvss",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  getNodeComponentSearchCategory(),
		Analyzer:  "",
	}
	nodeComponentObjVersionField = &search.Field{
		FieldPath: getNodeComponentPrefix() + ".version",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  getNodeComponentSearchCategory(),
		Analyzer:  "",
	}
	// Field Values - NodeComponentCVEEdge
	nodeComponentCVEEdgeFixableField = &search.Field{
		FieldPath: getNodeComponentVulnEdgePrefix() + ".is_fixable",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     true,
		Hidden:    false,
		Category:  getNodeComponentCVEEdgeSearchCategory(),
		Analyzer:  "",
	}
	nodeComponentCVEEdgeFixedByField = &search.Field{
		FieldPath: getNodeComponentVulnEdgePrefix() + ".HasFixedBy.fixed_by",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  getNodeComponentCVEEdgeSearchCategory(),
		Analyzer:  "",
	}
	// Field Values - NodeComponentEdge
	nodeComponentEdgeLocationField = &search.Field{
		FieldPath: getNodeComponentEdgePrefix() + ".location",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_NODE_COMPONENT_EDGE,
		Analyzer:  "",
	}
	// Field Values - NodeCVE
	nodeCVECreatedTimeField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".cve_base_info.created_at.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	nodeCVECVEField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".cve_base_info.cve",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	nodeCVECVSSField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".cvss",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	nodeCVEIDField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	nodeCVEImpactScoreField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".impact_score",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    false,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	nodeCVEOperatingSystemField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".operating_system",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	nodeCVEPublishedOnField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".cve_base_info.published_on.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	nodeCVESeverityField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".severity",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	nodeCVESnoozedField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".snoozed",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     false,
		Hidden:    false,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	nodeCVESnoozeExpiryField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".snooze_expiry.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    true,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	// Field Values - ProcessIndicator
	processIndicatorObjContainerIDField = &search.Field{
		FieldPath: getProcessIndicatorPrefix() + ".signal.container_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_PROCESS_INDICATORS,
		Analyzer:  "",
	}
	processIndicatorObjContainerNameField = &search.Field{
		FieldPath: getProcessIndicatorPrefix() + ".container_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_PROCESS_INDICATORS,
		Analyzer:  "",
	}
	processIndicatorObjPodIDField = &search.Field{
		FieldPath: getProcessIndicatorPrefix() + ".pod_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_PROCESS_INDICATORS,
		Analyzer:  "",
	}
	processIndicatorObjPodLabelField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".pod_labels",
		Type:      v1.SearchDataType_SEARCH_MAP,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	processIndicatorObjPodUIDField = &search.Field{
		FieldPath: getProcessIndicatorPrefix() + ".pod_uid",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_PROCESS_INDICATORS,
		Analyzer:  "",
	}
	processIndicatorObjProcessArgumentsField = &search.Field{
		FieldPath: getProcessIndicatorPrefix() + ".signal.args",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_PROCESS_INDICATORS,
		Analyzer:  "",
	}
	processIndicatorObjProcessExecPathField = &search.Field{
		FieldPath: getProcessIndicatorPrefix() + ".signal.exec_file_path",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_PROCESS_INDICATORS,
		Analyzer:  "",
	}
	processIndicatorObjProcessIDField = &search.Field{
		FieldPath: getProcessIndicatorPrefix() + ".id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_PROCESS_INDICATORS,
		Analyzer:  "",
	}
	processIndicatorObjProcessNameField = &search.Field{
		FieldPath: getProcessIndicatorPrefix() + ".signal.name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_PROCESS_INDICATORS,
		Analyzer:  "",
	}
	processIndicatorObjProcessUIDField = &search.Field{
		FieldPath: getProcessIndicatorPrefix() + ".signal.uid",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_PROCESS_INDICATORS,
		Analyzer:  "",
	}

	// Composite OptionsMaps
	postgresImageToVulnFieldMap = map[search.FieldLabel]*search.Field{
		search.AddCapabilities:               deploymentAddCapabilitiesField,
		search.AdmissionControlStatus:        clusterAdmissionControlStatusField,
		search.Cluster:                       deploymentClusterNameField,
		search.ClusterID:                     deploymentClusterIDField,
		search.ClusterLabel:                  deploymentClusterLabelField,
		search.ClusterStatus:                 clusterClusterStatusField,
		search.CollectorStatus:               clusterCollectorStatusField,
		search.Component:                     imageObjComponentNameField,
		search.ComponentCount:                imageObjComponentCountField,
		search.ComponentID:                   imageComponentObjIDField,
		search.ComponentLocation:             imageComponentEdgeLocationField,
		search.ComponentPriority:             imageComponentObjPriorityField,
		search.ComponentRiskScore:            imageObjComponentRiskScoreField,
		search.ComponentSource:               imageComponentObjSourceField,
		search.ComponentTopCVSS:              imageComponentObjTopCVSSField,
		search.ComponentVersion:              imageObjComponentVersionField,
		search.CPUCoresLimit:                 deploymentCPUCoresLimitField,
		search.CPUCoresRequest:               deploymentCPUCoresRequestField,
		search.Created:                       deploymentCreatedField,
		search.CVE:                           imageObjCVEField,
		search.CVECount:                      imageObjCVECountField,
		search.CVECreatedTime:                imageCVECreatedTimeField,
		search.CVEID:                         imageCVEIDField,
		search.CVEPublishedOn:                imageObjCVEPublishedOnField,
		search.CVESuppressed:                 imageObjCVESuppressedField,
		search.CVESuppressExpiry:             imageCVESnoozeExpiryField,
		search.CVSS:                          imageObjCVSSField,
		search.DeploymentAnnotation:          deploymentAnnotationsField,
		search.DeploymentID:                  deploymentIDField,
		search.DeploymentLabel:               deploymentLabelsField,
		search.DeploymentName:                deploymentNameField,
		search.DeploymentPriority:            deploymentPriorityField,
		search.DeploymentRiskScore:           deploymentRiskScoreField,
		search.DeploymentType:                deploymentTypeField,
		search.DockerfileInstructionKeyword:  imageObjDockerfileInstructionField,
		search.DockerfileInstructionValue:    imageObjDockerfileInstructionValueField,
		search.DropCapabilities:              deploymentDropCapabilitiesField,
		search.EnvironmentKey:                deploymentEnvKeyField,
		search.EnvironmentValue:              deploymentEnvValueField,
		search.EnvironmentVarSrc:             deploymentEnvVarSourceField,
		search.ExposedNodePort:               deploymentExposedNodePortField,
		search.ExposingService:               deploymentExposingServiceField,
		search.ExposingServicePort:           deploymentExposingServicePortField,
		search.ExposureLevel:                 deploymentExposureLevelField,
		search.ExternalHostname:              deploymentExternalHostnameField,
		search.ExternalIP:                    deploymentExternalIPField,
		search.FirstImageOccurrenceTimestamp: imageCVEEdgeFirstOccurrenceField,
		search.Fixable:                       imageComponentCVEEdgeFixableField,
		search.FixableCVECount:               imageObjFixableCVEsField,
		search.FixedBy:                       imageObjFixedByField,
		search.ImageCommand:                  imageObjCommandField,
		search.ImageCreatedTime:              imageObjCreatedTimeField,
		search.ImageEntrypoint:               imageObjEntrypointField,
		search.ImageLabel:                    imageObjLabelField,
		search.ImageName:                     imageObjNameField,
		search.ImageOS:                       imageObjOperatingSystemField,
		search.ImagePriority:                 imageObjPriorityField,
		search.ImagePullSecret:               deploymentImagePullSecretsField,
		search.ImageRegistry:                 imageObjRegistryField,
		search.ImageRemote:                   imageObjRemoteField,
		search.ImageRiskScore:                imageObjRiskScoreField,
		search.ImageScanTime:                 imageObjScanTimeField,
		search.ImageSHA:                      imageObjIDField,
		search.ImageSignatureFetchedTime:     imageObjSignatureFetchTimeField,
		search.ImageTag:                      imageObjTagField,
		search.ImageTopCVSS:                  imageObjTopCVSSField,
		search.ImageUser:                     imageObjUserField,
		search.ImageVolumes:                  imageObjVolumesField,
		search.ImpactScore:                   imageCVEImpactScoreField,
		search.LastContactTime:               clusterLastContactField,
		search.LastUpdatedTime:               imageObjLastUpdatedField,
		search.MaxExposureLevel:              deploymentMaxExposureField,
		search.MemoryLimit:                   deploymentMemoryLimitField,
		search.MemoryRequest:                 deploymentMemoryRequestField,
		search.Namespace:                     deploymentNamespaceField,
		search.NamespaceAnnotation:           namespaceAnnotationsField,
		search.NamespaceID:                   deploymentNamespaceIDField,
		search.NamespaceLabel:                namespaceLabelField,
		search.OperatingSystem:               imageComponentObjOperatingSystemField,
		search.OrchestratorComponent:         deploymentOrchestratorComponentField,
		search.PodLabel:                      deploymentPodLabelField,
		search.Port:                          deploymentPortField,
		search.PortProtocol:                  deploymentPortProtocolField,
		search.Privileged:                    deploymentPrivilegedField,
		search.ReadOnlyRootFilesystem:        deploymentReadOnlyRootFilesystemField,
		search.ScannerStatus:                 clusterScannerStatusField,
		search.SecretName:                    deploymentSecretNameField,
		search.SecretPath:                    deploymentSecretPathField,
		search.SensorStatus:                  clusterSensorStatusField,
		search.ServiceAccountName:            deploymentServiceAccountNameField,
		search.ServiceAccountPermissionLevel: deploymentServiceAccountPermissionLevelField,
		search.Severity:                      imageCVESeverityField,
		search.VolumeDestination:             deploymentVolumeDestinationField,
		search.VolumeName:                    deploymentVolumeNameField,
		search.VolumeReadonly:                deploymentVolumeReadOnlyField,
		search.VolumeSource:                  deploymentVolumeSourceField,
		search.VolumeType:                    deploymentVolumeTypeField,
		search.VulnerabilityState:            imageObjCVEStateField,
	}
	postgresNodeToVulnFieldMap = map[search.FieldLabel]*search.Field{
		search.AdmissionControlStatus: clusterAdmissionControlStatusField,
		search.Cluster:                nodeObjClusterNameField,
		search.ClusterID:              nodeObjClusterIDField,
		search.ClusterLabel:           clusterLabelsField,
		search.ClusterStatus:          clusterClusterStatusField,
		search.CollectorStatus:        clusterCollectorStatusField,
		search.Component:              nodeObjComponentField,
		search.ComponentCount:         nodeObjComponentCountField,
		search.ComponentRiskScore:     nodeComponentObjRiskScoreField,
		search.ComponentTopCVSS:       nodeComponentObjTopCVSSField,
		search.ContainerRuntime:       nodeObjContainerRuntimeVersionField,
		search.ComponentID:            nodeComponentObjIDField,
		search.ComponentPriority:      nodeComponentObjPriorityField,
		search.ComponentVersion:       nodeObjComponentVersionField,
		search.CVE:                    nodeObjCVEField,
		search.CVECount:               nodeObjCVECountField,
		search.CVECreatedTime:         nodeObjCVECreatedTimeField,
		search.CVEID:                  nodeCVEIDField,
		search.CVEPublishedOn:         nodeObjCVEPublishedOnField,
		search.CVESuppressed:          nodeObjCVESnoozedField,
		search.CVESuppressExpiry:      nodeCVESnoozeExpiryField,
		search.CVSS:                   nodeObjCVSSField,
		search.Fixable:                nodeComponentCVEEdgeFixableField,
		search.FixableCVECount:        nodeObjFixableCVECountField,
		search.FixedBy:                nodeObjFixedByField,
		search.ImpactScore:            nodeCVEImpactScoreField,
		search.LastContactTime:        clusterLastContactField,
		search.LastUpdatedTime:        nodeObjLastUpdatedField,
		search.Node:                   nodeObjNameField,
		search.NodeAnnotation:         nodeObjAnnotationField,
		search.NodeID:                 nodeObjIDField,
		search.NodeJoinTime:           nodeObjJoinTimeField,
		search.NodeLabel:              nodeObjLabelField,
		search.NodePriority:           nodeObjNodeRiskPriorityField,
		search.NodeRiskScore:          nodeObjRiskScoreField,
		search.NodeScanTime:           nodeObjNodeScanTimeField,
		search.NodeTopCVSS:            nodeObjTopCVSSField,
		search.OperatingSystem:        nodeObjOperatingSystemField,
		search.ScannerStatus:          clusterScannerStatusField,
		search.SensorStatus:           clusterSensorStatusField,
		search.Severity:               nodeCVESeverityField,
		search.TaintKey:               nodeObjTaintKeyField,
		search.TaintValue:             nodeObjTaintValueField,
		search.TolerationEffect:       nodeObjTaintEffectField,
		search.VulnerabilityState:     nodeObjVulnerabilityStateField,
	}

	// Field Values - WIP
)

func TestActiveComponentMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.ComponentID:   activeComponentIDField,
		search.ContainerName: activeComponentContainerNameField,
		search.DeploymentID:  activeComponentDeploymentIDField,
		search.ImageSHA:      activeComponentImageIDField,
	}
	validateOptionsMap(t,
		v1.SearchCategory_ACTIVE_COMPONENT,
		expectedSearchFieldMap,
		nil,
		nil)
}

func TestAlertMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Category:       alertCategoryField,
		search.DeploymentID:   alertDeploymentIDField,
		search.DeploymentName: alertDeploymentNameField,
		search.Inactive:       alertInactiveField,
		search.PolicyID:       alertPolicyIDField,
		search.PolicyName:     alertPolicyNameField,
		search.ResourceName:   alertResourceNameField,
		search.Severity:       alertPolicySeverityField,
		search.ViolationState: alertStateField,
		search.ViolationTime:  alertViolationTimeField,
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Cluster:        alertClusterNameLegacyField,
		search.ClusterID:      alertClusterIDLegacyField,
		search.Enforcement:    alertEnforcementLegacyField,
		search.LifecycleStage: alertLifecycleStageLegacyField,
		search.Namespace:      alertNamespaceNameLegacyField,
		search.NamespaceID:    alertNamespaceIDLegacyField,
		search.ResourceType:   alertResourceTypeLegacyField,
		search.SORTPolicyName: alertSortPolicyNameLegacyField,
	}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Cluster:        alertClusterNamePostgresField,
		search.ClusterID:      alertClusterIDPostgresField,
		search.Enforcement:    alertEnforcementPostgresField,
		search.LifecycleStage: alertLifecycleStagePostgresField,
		search.Namespace:      alertNamespaceNamePostgresField,
		search.NamespaceID:    alertNamespaceIDPostgresField,
		search.ResourceType:   alertResourceTypePostgresField,
		search.SORTPolicyName: alertSortPolicyNamePostgresField,
	}
	validateOptionsMap(t,
		v1.SearchCategory_ALERTS,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestClusterVulnEdgeMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.ClusterCVEFixable: clusterCVEEdgeFixableField,
		search.ClusterCVEFixedBy: clusterCVEEdgeFixedByField,
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.AdmissionControlStatus: clusterAdmissionControlStatusField,
		search.Cluster:                clusterNameField,
		search.ClusterID:              clusterIDField,
		search.ClusterLabel:           clusterLabelsField,
		search.ClusterStatus:          clusterClusterStatusField,
		search.CollectorStatus:        clusterCollectorStatusField,
		search.CVE:                    clusterCVECVEField,
		search.CVECreatedTime:         clusterCVECreatedTimeField,
		search.CVEID:                  clusterCVEIDField,
		search.CVEPublishedOn:         clusterCVEPublishedOnField,
		search.CVESuppressed:          clusterCVESnoozedField,
		search.CVESuppressExpiry:      clusterCVESnoozeExpiryField,
		search.CVEType:                clusterCVETypeField,
		search.CVSS:                   clusterCVECVSSField,
		search.ImpactScore:            clusterCVEImpactScore,
		search.LastContactTime:        clusterLastContactField,
		search.ScannerStatus:          clusterScannerStatusField,
		search.SensorStatus:           clusterSensorStatusField,
		search.Severity:               ClusterCVESeverityField,
	}
	validateOptionsMap(t,
		v1.SearchCategory_CLUSTER_VULN_EDGE,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestClusterVulnerabilitiesMapping(t *testing.T) {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		return
	}
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.AdmissionControlStatus: clusterAdmissionControlStatusField,
		search.Cluster:                clusterNameField,
		search.ClusterCVEFixable:      clusterCVEEdgeFixableField,
		search.ClusterCVEFixedBy:      clusterCVEEdgeFixedByField,
		search.ClusterID:              clusterIDField,
		search.ClusterLabel:           clusterLabelsField,
		search.ClusterStatus:          clusterClusterStatusField,
		search.CollectorStatus:        clusterCollectorStatusField,
		search.CVE:                    clusterCVECVEField,
		search.CVECreatedTime:         clusterCVECreatedTimeField,
		search.CVEID:                  clusterCVEIDField,
		search.CVEPublishedOn:         clusterCVEPublishedOnField,
		search.CVESuppressed:          clusterCVESnoozedField,
		search.CVESuppressExpiry:      clusterCVESnoozeExpiryField,
		search.CVEType:                clusterCVETypeField,
		search.CVSS:                   clusterCVECVSSField,
		search.ImpactScore:            clusterCVEImpactScore,
		search.LastContactTime:        clusterLastContactField,
		search.ScannerStatus:          clusterScannerStatusField,
		search.SensorStatus:           clusterSensorStatusField,
		search.Severity:               ClusterCVESeverityField,
	}
	validateOptionsMap(t,
		v1.SearchCategory_CLUSTER_VULNERABILITIES,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestClustersMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.AdmissionControlStatus: clusterAdmissionControlStatusField,
		search.Cluster:                clusterNameField,
		search.ClusterID:              clusterIDField,
		search.ClusterLabel:           clusterLabelsField,
		search.ClusterStatus:          clusterClusterStatusField,
		search.CollectorStatus:        clusterCollectorStatusField,
		search.LastContactTime:        clusterLastContactField,
		search.ScannerStatus:          clusterScannerStatusField,
		search.SensorStatus:           clusterSensorStatusField,
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	validateOptionsMap(t,
		v1.SearchCategory_CLUSTERS,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)

}

func TestComplianceControlMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Control:        complianceControlNameField,
		search.ControlID:      complianceControlIDField,
		search.ControlGroupID: complianceControlGroupIDField,
		search.StandardID:     complianceControlStandardIDField,
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	validateOptionsMap(t,
		v1.SearchCategory_COMPLIANCE_CONTROL,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestComplianceStandardMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Standard:   complianceStandardNameField,
		search.StandardID: complianceStandardIDField,
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	validateOptionsMap(t,
		v1.SearchCategory_COMPLIANCE_STANDARD,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestComponentVulnEdgeMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Fixable: imageComponentCVEEdgeFixableField,
		search.FixedBy: imageComponentCVEEdgeFixedByField,
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	validateOptionsMap(t,
		v1.SearchCategory_COMPONENT_VULN_EDGE,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestDeploymentMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.AddCapabilities:               deploymentAddCapabilitiesField,
		search.Cluster:                       deploymentClusterNameField,
		search.ClusterID:                     deploymentClusterIDField,
		search.Component:                     imageObjComponentNameField,
		search.ComponentCount:                imageObjComponentCountField,
		search.ComponentRiskScore:            imageObjComponentRiskScoreField,
		search.ComponentVersion:              imageObjComponentVersionField,
		search.ContainerID:                   processIndicatorObjContainerIDField,
		search.ContainerName:                 processIndicatorObjContainerNameField,
		search.CPUCoresLimit:                 deploymentCPUCoresLimitField,
		search.CPUCoresRequest:               deploymentCPUCoresRequestField,
		search.Created:                       deploymentCreatedField,
		search.CVE:                           imageObjCVEField,
		search.CVECount:                      imageObjCVECountField,
		search.CVEPublishedOn:                imageObjCVEPublishedOnField,
		search.CVESuppressed:                 imageObjCVESuppressedField,
		search.CVSS:                          imageObjCVSSField,
		search.DeploymentAnnotation:          deploymentAnnotationsField,
		search.DeploymentID:                  deploymentIDField,
		search.DeploymentLabel:               deploymentLabelsField,
		search.DeploymentName:                deploymentNameField,
		search.DeploymentPriority:            deploymentPriorityField,
		search.DeploymentRiskScore:           deploymentRiskScoreField,
		search.DeploymentType:                deploymentTypeField,
		search.DockerfileInstructionKeyword:  imageObjDockerfileInstructionField,
		search.DockerfileInstructionValue:    imageObjDockerfileInstructionValueField,
		search.DropCapabilities:              deploymentDropCapabilitiesField,
		search.EnvironmentKey:                deploymentEnvKeyField,
		search.EnvironmentValue:              deploymentEnvValueField,
		search.EnvironmentVarSrc:             deploymentEnvVarSourceField,
		search.ExposedNodePort:               deploymentExposedNodePortField,
		search.ExposingService:               deploymentExposingServiceField,
		search.ExposingServicePort:           deploymentExposingServicePortField,
		search.ExposureLevel:                 deploymentExposureLevelField,
		search.ExternalHostname:              deploymentExternalHostnameField,
		search.ExternalIP:                    deploymentExternalIPField,
		search.FixableCVECount:               imageObjFixableCVEsField,
		search.FixedBy:                       imageObjFixedByField,
		search.ImageCommand:                  imageObjCommandField,
		search.ImageCreatedTime:              imageObjCreatedTimeField,
		search.ImageEntrypoint:               imageObjEntrypointField,
		search.ImageLabel:                    imageObjLabelField,
		search.ImageName:                     deploymentImageNameField,
		search.ImageOS:                       imageObjOperatingSystemField,
		search.ImagePriority:                 imageObjPriorityField,
		search.ImagePullSecret:               deploymentImagePullSecretsField,
		search.ImageRegistry:                 deploymentImageRegistryField,
		search.ImageRemote:                   deploymentImageRemoteField,
		search.ImageRiskScore:                imageObjRiskScoreField,
		search.ImageScanTime:                 imageObjScanTimeField,
		search.ImageSHA:                      deploymentImageIDField,
		search.ImageSignatureFetchedTime:     imageObjSignatureFetchTimeField,
		search.ImageTag:                      deploymentImageTagField,
		search.ImageTopCVSS:                  imageObjTopCVSSField,
		search.ImageUser:                     imageObjUserField,
		search.ImageVolumes:                  imageObjVolumesField,
		search.LastUpdatedTime:               imageObjLastUpdatedField,
		search.MaxExposureLevel:              deploymentMaxExposureField,
		search.MemoryLimit:                   deploymentMemoryLimitField,
		search.MemoryRequest:                 deploymentMemoryRequestField,
		search.Namespace:                     deploymentNamespaceField,
		search.NamespaceID:                   deploymentNamespaceIDField,
		search.OrchestratorComponent:         deploymentOrchestratorComponentField,
		search.PodID:                         processIndicatorObjPodIDField,
		search.PodLabel:                      processIndicatorObjPodLabelField,
		search.PodUID:                        processIndicatorObjPodUIDField,
		search.Port:                          deploymentPortField,
		search.PortProtocol:                  deploymentPortProtocolField,
		search.Privileged:                    deploymentPrivilegedField,
		search.ProcessArguments:              processIndicatorObjProcessArgumentsField,
		search.ProcessExecPath:               processIndicatorObjProcessExecPathField,
		search.ProcessID:                     processIndicatorObjProcessIDField,
		search.ProcessName:                   processIndicatorObjProcessNameField,
		search.ProcessUID:                    processIndicatorObjProcessUIDField,
		search.ReadOnlyRootFilesystem:        deploymentReadOnlyRootFilesystemField,
		search.SecretName:                    deploymentSecretNameField,
		search.SecretPath:                    deploymentSecretPathField,
		search.ServiceAccountName:            deploymentServiceAccountNameField,
		search.ServiceAccountPermissionLevel: deploymentServiceAccountPermissionLevelField,
		search.VolumeDestination:             deploymentVolumeDestinationField,
		search.VolumeName:                    deploymentVolumeNameField,
		search.VolumeReadonly:                deploymentVolumeReadOnlyField,
		search.VolumeSource:                  deploymentVolumeSourceField,
		search.VolumeType:                    deploymentVolumeTypeField,
		search.VulnerabilityState:            imageObjCVEStateField,
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	validateOptionsMap(t,
		v1.SearchCategory_DEPLOYMENTS,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestImageMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Cluster:                      deploymentClusterNameField,
		search.ClusterID:                    deploymentClusterIDField,
		search.Component:                    imageObjComponentNameField,
		search.ComponentCount:               imageObjComponentCountField,
		search.ComponentID:                  imageComponentObjIDField,
		search.ComponentLocation:            imageComponentEdgeLocationField,
		search.ComponentPriority:            imageComponentObjPriorityField,
		search.ComponentRiskScore:           imageObjComponentRiskScoreField,
		search.ComponentSource:              imageComponentObjSourceField,
		search.ComponentTopCVSS:             imageComponentObjTopCVSSField,
		search.ComponentVersion:             imageObjComponentVersionField,
		search.CVE:                          imageObjCVEField,
		search.CVECount:                     imageObjCVECountField,
		search.CVEPublishedOn:               imageObjCVEPublishedOnField,
		search.CVESuppressed:                imageObjCVESuppressedField,
		search.CVSS:                         imageObjCVSSField,
		search.DeploymentID:                 deploymentIDField,
		search.DeploymentLabel:              deploymentLabelsField,
		search.DeploymentName:               deploymentNameField,
		search.DockerfileInstructionKeyword: imageObjDockerfileInstructionField,
		search.DockerfileInstructionValue:   imageObjDockerfileInstructionValueField,
		search.Fixable:                      imageComponentCVEEdgeFixableField,
		search.FixableCVECount:              imageObjFixableCVEsField,
		search.FixedBy:                      imageObjFixedByField,
		search.ImageCommand:                 imageObjCommandField,
		search.ImageCreatedTime:             imageObjCreatedTimeField,
		search.ImageEntrypoint:              imageObjEntrypointField,
		search.ImageLabel:                   imageObjLabelField,
		search.ImageName:                    imageObjNameField,
		search.ImageOS:                      imageObjOperatingSystemField,
		search.ImagePriority:                imageObjPriorityField,
		search.ImageRegistry:                imageObjRegistryField,
		search.ImageRemote:                  imageObjRemoteField,
		search.ImageRiskScore:               imageObjRiskScoreField,
		search.ImageScanTime:                imageObjScanTimeField,
		search.ImageSHA:                     imageObjIDField,
		search.ImageSignatureFetchedTime:    imageObjSignatureFetchTimeField,
		search.ImageTag:                     imageObjTagField,
		search.ImageTopCVSS:                 imageObjTopCVSSField,
		search.ImageUser:                    imageObjUserField,
		search.ImageVolumes:                 imageObjVolumesField,
		search.LastUpdatedTime:              imageObjLastUpdatedField,
		search.Namespace:                    deploymentNamespaceField,
		search.NamespaceID:                  deploymentNamespaceIDField,
		search.OperatingSystem:              imageComponentObjOperatingSystemField,
		search.VulnerabilityState:           imageObjCVEStateField,
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{
		search.CVECreatedTime:    cveLegacyObjCreatedTimeField,
		search.CVESuppressExpiry: cveLegacyObjSnoozeExpiryField,
		search.CVEType:           cveLegacyObjTypeField,
		search.ImpactScore:       cveLegacyObjImpactScoreField,
		search.Severity:          cveLegacyObjSeverityField,
	}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	for k, v := range postgresImageToVulnFieldMap {
		if _, found := expectedSearchFieldMap[k]; !found {
			expectedPostgresSearchFieldMap[k] = v
		}
	}
	validateOptionsMap(t,
		v1.SearchCategory_IMAGES,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestImageComponentMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Cluster:                      deploymentClusterNameField,
		search.ClusterID:                    deploymentClusterIDField,
		search.Component:                    imageComponentObjNameField,
		search.ComponentCount:               imageObjComponentCountField,
		search.ComponentID:                  imageComponentObjIDField,
		search.ComponentPriority:            imageComponentObjPriorityField,
		search.ComponentLocation:            imageComponentEdgeLocationField,
		search.ComponentRiskScore:           imageComponentObjRiskScoreField,
		search.ComponentSource:              imageComponentObjSourceField,
		search.ComponentTopCVSS:             imageComponentObjTopCVSSField,
		search.ComponentVersion:             imageComponentObjVersionField,
		search.CVE:                          imageObjCVEField,
		search.CVECount:                     imageObjCVECountField,
		search.CVEPublishedOn:               imageObjCVEPublishedOnField,
		search.CVESuppressed:                imageObjCVESuppressedField,
		search.CVSS:                         imageObjCVSSField,
		search.DeploymentID:                 deploymentIDField,
		search.DeploymentLabel:              deploymentLabelsField,
		search.DeploymentName:               deploymentNameField,
		search.DockerfileInstructionKeyword: imageObjDockerfileInstructionField,
		search.DockerfileInstructionValue:   imageObjDockerfileInstructionValueField,
		search.Fixable:                      imageComponentCVEEdgeFixableField,
		search.FixableCVECount:              imageObjFixableCVEsField,
		search.FixedBy:                      imageComponentCVEEdgeFixedByField,
		search.ImageCommand:                 imageObjCommandField,
		search.ImageCreatedTime:             imageObjCreatedTimeField,
		search.ImageEntrypoint:              imageObjEntrypointField,
		search.ImageLabel:                   imageObjLabelField,
		search.ImageName:                    imageObjNameField,
		search.ImageOS:                      imageObjOperatingSystemField,
		search.ImagePriority:                imageObjPriorityField,
		search.ImageRegistry:                imageObjRegistryField,
		search.ImageRemote:                  imageObjRemoteField,
		search.ImageRiskScore:               imageObjRiskScoreField,
		search.ImageScanTime:                imageObjScanTimeField,
		search.ImageSHA:                     imageObjIDField,
		search.ImageSignatureFetchedTime:    imageObjSignatureFetchTimeField,
		search.ImageTag:                     imageObjTagField,
		search.ImageTopCVSS:                 imageObjTopCVSSField,
		search.ImageUser:                    imageObjUserField,
		search.ImageVolumes:                 imageObjVolumesField,
		search.ImpactScore:                  imageCVEImpactScoreField,
		search.LastUpdatedTime:              imageObjLastUpdatedField,
		search.Namespace:                    deploymentNamespaceField,
		search.NamespaceID:                  deploymentNamespaceIDField,
		search.OperatingSystem:              imageComponentObjOperatingSystemField,
		search.Severity:                     imageCVESeverityField,
		search.VulnerabilityState:           imageObjCVEStateField,
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{
		search.CVECreatedTime:    cveLegacyObjCreatedTimeField,
		search.CVESuppressExpiry: cveLegacyObjSnoozeExpiryField,
		search.CVEType:           cveLegacyObjTypeField,
	}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	for k, v := range postgresImageToVulnFieldMap {
		if _, found := expectedSearchFieldMap[k]; !found {
			expectedPostgresSearchFieldMap[k] = v
		}
	}
	expectedPostgresSearchFieldMap[search.CVECreatedTime] = imageCVECreatedTimeField
	expectedPostgresSearchFieldMap[search.CVESuppressExpiry] = imageCVESnoozeExpiryField
	validateOptionsMap(t,
		v1.SearchCategory_IMAGE_COMPONENTS,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestImageComponentEdgeMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.ComponentLocation: imageComponentEdgeLocationField,
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	for k, v := range postgresImageToVulnFieldMap {
		if _, found := expectedSearchFieldMap[k]; !found {
			expectedPostgresSearchFieldMap[k] = v
		}
	}
	validateOptionsMap(t,
		v1.SearchCategory_IMAGE_COMPONENT_EDGE,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestImageCVEMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.AddCapabilities:               deploymentAddCapabilitiesField,
		search.Cluster:                       deploymentClusterNameField,
		search.ClusterID:                     deploymentClusterIDField,
		search.Component:                     imageComponentObjNameField,
		search.ComponentCount:                imageObjComponentCountField,
		search.ComponentID:                   imageComponentObjIDField,
		search.ComponentLocation:             imageComponentEdgeLocationField,
		search.ComponentPriority:             imageComponentObjPriorityField,
		search.ComponentRiskScore:            imageComponentObjRiskScoreField,
		search.ComponentSource:               imageComponentObjSourceField,
		search.ComponentTopCVSS:              imageComponentObjTopCVSSField,
		search.ComponentVersion:              imageComponentObjVersionField,
		search.ContainerID:                   processIndicatorObjContainerIDField,
		search.ContainerName:                 processIndicatorObjContainerNameField,
		search.CPUCoresLimit:                 deploymentCPUCoresLimitField,
		search.CPUCoresRequest:               deploymentCPUCoresRequestField,
		search.Created:                       deploymentCreatedField,
		search.CVECount:                      imageObjCVECountField,
		search.DeploymentAnnotation:          deploymentAnnotationsField,
		search.DeploymentID:                  deploymentIDField,
		search.DeploymentLabel:               deploymentLabelsField,
		search.DeploymentName:                deploymentNameField,
		search.DeploymentPriority:            deploymentPriorityField,
		search.DeploymentRiskScore:           deploymentRiskScoreField,
		search.DeploymentType:                deploymentTypeField,
		search.DockerfileInstructionKeyword:  imageObjDockerfileInstructionField,
		search.DockerfileInstructionValue:    imageObjDockerfileInstructionValueField,
		search.DropCapabilities:              deploymentDropCapabilitiesField,
		search.EnvironmentKey:                deploymentEnvKeyField,
		search.EnvironmentValue:              deploymentEnvValueField,
		search.EnvironmentVarSrc:             deploymentEnvVarSourceField,
		search.ExposedNodePort:               deploymentExposedNodePortField,
		search.ExposingService:               deploymentExposingServiceField,
		search.ExposingServicePort:           deploymentExposingServicePortField,
		search.ExposureLevel:                 deploymentExposureLevelField,
		search.ExternalHostname:              deploymentExternalHostnameField,
		search.ExternalIP:                    deploymentExternalIPField,
		search.Fixable:                       imageComponentCVEEdgeFixableField,
		search.FixableCVECount:               imageObjFixableCVEsField,
		search.FixedBy:                       imageComponentCVEEdgeFixedByField,
		search.ImageCommand:                  imageObjCommandField,
		search.ImageCreatedTime:              imageObjCreatedTimeField,
		search.ImageEntrypoint:               imageObjEntrypointField,
		search.ImageLabel:                    imageObjLabelField,
		search.ImageName:                     imageObjNameField,
		search.ImageOS:                       imageObjOperatingSystemField,
		search.ImagePriority:                 imageObjPriorityField,
		search.ImagePullSecret:               deploymentImagePullSecretsField,
		search.ImageRegistry:                 imageObjRegistryField,
		search.ImageRemote:                   imageObjRemoteField,
		search.ImageRiskScore:                imageObjRiskScoreField,
		search.ImageScanTime:                 imageObjScanTimeField,
		search.ImageSHA:                      imageObjIDField,
		search.ImageSignatureFetchedTime:     imageObjSignatureFetchTimeField,
		search.ImageTag:                      imageObjTagField,
		search.ImageTopCVSS:                  imageObjTopCVSSField,
		search.ImageUser:                     imageObjUserField,
		search.ImageVolumes:                  imageObjVolumesField,
		search.LastUpdatedTime:               imageObjLastUpdatedField,
		search.MaxExposureLevel:              deploymentMaxExposureField,
		search.MemoryLimit:                   deploymentMemoryLimitField,
		search.MemoryRequest:                 deploymentMemoryRequestField,
		search.Namespace:                     deploymentNamespaceField,
		search.NamespaceID:                   deploymentNamespaceIDField,
		search.OrchestratorComponent:         deploymentOrchestratorComponentField,
		search.PodLabel:                      deploymentPodLabelField,
		search.PodID:                         processIndicatorObjPodIDField,
		search.PodUID:                        processIndicatorObjPodUIDField,
		search.Port:                          deploymentPortField,
		search.PortProtocol:                  deploymentPortProtocolField,
		search.Privileged:                    deploymentPrivilegedField,
		search.ProcessArguments:              processIndicatorObjProcessArgumentsField,
		search.ProcessExecPath:               processIndicatorObjProcessExecPathField,
		search.ProcessID:                     processIndicatorObjProcessIDField,
		search.ProcessName:                   processIndicatorObjProcessNameField,
		search.ProcessUID:                    processIndicatorObjProcessUIDField,
		search.ReadOnlyRootFilesystem:        deploymentReadOnlyRootFilesystemField,
		search.SecretName:                    deploymentSecretNameField,
		search.SecretPath:                    deploymentSecretPathField,
		search.ServiceAccountName:            deploymentServiceAccountNameField,
		search.ServiceAccountPermissionLevel: deploymentServiceAccountPermissionLevelField,
		search.VolumeDestination:             deploymentVolumeDestinationField,
		search.VolumeName:                    deploymentVolumeNameField,
		search.VolumeReadonly:                deploymentVolumeReadOnlyField,
		search.VolumeSource:                  deploymentVolumeSourceField,
		search.VolumeType:                    deploymentVolumeTypeField,
		search.VulnerabilityState:            imageObjCVEStateField,
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{
		search.CVE:               cveLegacyObjIDField,
		search.CVECreatedTime:    cveLegacyObjCreatedTimeField,
		search.CVEPublishedOn:    cveLegacyObjPublishedOnField,
		search.CVESuppressed:     cveLegacyObjSnoozedField,
		search.CVESuppressExpiry: cveLegacyObjSnoozeExpiryField,
		search.CVEType:           cveLegacyObjTypeField,
		search.CVSS:              cveLegacyObjCVSSField,
		search.ImpactScore:       cveLegacyObjImpactScoreField,
		search.OperatingSystem:   imageComponentObjOperatingSystemField,
		search.Severity:          cveLegacyObjSeverityField,
	}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.AdmissionControlStatus:        clusterAdmissionControlStatusField,
		search.ClusterLabel:                  clusterLabelsField,
		search.ClusterStatus:                 clusterClusterStatusField,
		search.CollectorStatus:               clusterCollectorStatusField,
		search.CVE:                           imageCVECVEField,
		search.CVECreatedTime:                imageCVECreatedTimeField,
		search.CVEID:                         imageCVEIDField,
		search.CVEPublishedOn:                imageCVEPublishedOnField,
		search.CVESuppressed:                 imageCVESnoozedField,
		search.CVESuppressExpiry:             imageCVESnoozeExpiryField,
		search.CVSS:                          imageCVECVSSField,
		search.FirstImageOccurrenceTimestamp: imageCVEEdgeFirstOccurrenceField,
		search.ImpactScore:                   imageCVEImpactScoreField,
		search.LastContactTime:               clusterLastContactField,
		search.NamespaceAnnotation:           namespaceAnnotationsField,
		search.NamespaceLabel:                namespaceLabelField,
		search.OperatingSystem:               imageCVEOperatingSystemField,
		search.ScannerStatus:                 clusterScannerStatusField,
		search.SensorStatus:                  clusterSensorStatusField,
		search.Severity:                      imageCVESeverityField,
	}
	validateOptionsMap(t,
		getImageCVESearchCategory(),
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestImageCVEEdgeMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.FirstImageOccurrenceTimestamp: imageCVEEdgeFirstOccurrenceField,
		search.VulnerabilityState:            imageCVEEdgeStateField,
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	validateOptionsMap(t,
		v1.SearchCategory_IMAGE_VULN_EDGE,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestImageIntegrationsMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.ClusterID: imageIntegrationObjClusterIDField,
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	validateOptionsMap(t,
		v1.SearchCategory_IMAGE_INTEGRATIONS,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestNamespaceMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Cluster:             namespaceClusterField,
		search.ClusterID:           namespaceClusterIDField,
		search.Namespace:           namespaceNameField,
		search.NamespaceAnnotation: namespaceAnnotationsField,
		search.NamespaceID:         namespaceIDField,
		search.NamespaceLabel:      namespaceLabelField,
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	validateOptionsMap(t,
		v1.SearchCategory_NAMESPACES,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestNodeMapping(t *testing.T) {
	targetMap := GetEntityOptionsMap()[v1.SearchCategory_NODES]
	for k, v := range targetMap.Original() {
		fmt.Println(k)
		fmt.Println(v)
	}
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Cluster:            nodeObjClusterNameField,
		search.ClusterID:          nodeObjClusterIDField,
		search.Component:          nodeObjComponentField,
		search.ComponentCount:     nodeObjComponentCountField,
		search.ComponentID:        nodeComponentObjIDField,
		search.ComponentPriority:  nodeComponentObjPriorityField,
		search.ComponentRiskScore: nodeComponentObjRiskScoreField,
		search.ComponentTopCVSS:   nodeComponentObjTopCVSSField,
		search.ComponentVersion:   nodeObjComponentVersionField,
		search.ContainerRuntime:   nodeObjContainerRuntimeVersionField,
		search.CVE:                nodeObjCVEField,
		search.CVECount:           nodeObjCVECountField,
		search.CVECreatedTime:     nodeObjCVECreatedTimeField,
		search.CVEPublishedOn:     nodeObjCVEPublishedOnField,
		search.CVESuppressed:      nodeObjCVESnoozedField,
		search.CVSS:               nodeObjCVSSField,
		search.Fixable:            nodeComponentCVEEdgeFixableField,
		search.FixableCVECount:    nodeObjFixableCVECountField,
		search.FixedBy:            nodeObjFixedByField,
		search.LastUpdatedTime:    nodeObjLastUpdatedField,
		search.Node:               nodeObjNameField,
		search.NodeAnnotation:     nodeObjAnnotationField,
		search.NodeID:             nodeObjIDField,
		search.NodeJoinTime:       nodeObjJoinTimeField,
		search.NodeLabel:          nodeObjLabelField,
		search.NodePriority:       nodeObjNodeRiskPriorityField,
		search.NodeRiskScore:      nodeObjRiskScoreField,
		search.NodeScanTime:       nodeObjNodeScanTimeField,
		search.NodeTopCVSS:        nodeObjTopCVSSField,
		search.OperatingSystem:    nodeObjOperatingSystemField,
		search.TaintKey:           nodeObjTaintKeyField,
		search.TaintValue:         nodeObjTaintValueField,
		search.TolerationEffect:   nodeObjTaintEffectField,
		search.VulnerabilityState: nodeObjVulnerabilityStateField,
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{
		search.ComponentSource:   nodeComponentObjSourceField,
		search.CVESuppressExpiry: cveLegacyObjSnoozeExpiryField,
		search.CVEType:           cveLegacyObjTypeField,
		search.ImpactScore:       cveLegacyObjImpactScoreField,
		search.Severity:          cveLegacyObjSeverityField,
	}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	for k, v := range postgresNodeToVulnFieldMap {
		if _, found := expectedSearchFieldMap[k]; !found {
			expectedPostgresSearchFieldMap[k] = v
		}
	}
	validateOptionsMap(t,
		v1.SearchCategory_NODES,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestNodeComponentMapping(t *testing.T) {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		return
	}
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	for k, v := range postgresNodeToVulnFieldMap {
		if _, found := expectedSearchFieldMap[k]; !found {
			expectedPostgresSearchFieldMap[k] = v
		}
	}
	validateOptionsMap(t,
		v1.SearchCategory_NODE_COMPONENTS,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestNodeComponentEdgeMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	for k, v := range postgresNodeToVulnFieldMap {
		if _, found := expectedSearchFieldMap[k]; !found {
			expectedPostgresSearchFieldMap[k] = v
		}
	}
	validateOptionsMap(t,
		v1.SearchCategory_NODE_COMPONENT_EDGE,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestNodeCVEMapping(t *testing.T) {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		return
	}
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{}
	for k, v := range postgresNodeToVulnFieldMap {
		if _, found := expectedSearchFieldMap[k]; !found {
			expectedPostgresSearchFieldMap[k] = v
		}
	}
	validateOptionsMap(t,
		v1.SearchCategory_NODE_VULNERABILITIES,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func validateOptionsMap(
	t *testing.T,
	category v1.SearchCategory,
	expectedFieldMap map[search.FieldLabel]*search.Field,
	expectedLegacyAddFieldMap map[search.FieldLabel]*search.Field,
	expectedPostgresAddFieldMap map[search.FieldLabel]*search.Field) {
	// Extract OptionsMap registered in the global index mapping for the search category
	actualMap := GetEntityOptionsMap()[category]
	expectedLen := len(expectedFieldMap) + len(expectedLegacyAddFieldMap) + len(expectedPostgresAddFieldMap)
	expectedSearchFieldLabels := make([]search.FieldLabel, 0, expectedLen)
	for k := range expectedFieldMap {
		expectedSearchFieldLabels = append(expectedSearchFieldLabels, k)
	}
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		for k := range expectedPostgresAddFieldMap {
			expectedSearchFieldLabels = append(expectedSearchFieldLabels, k)
		}
	} else {
		for k := range expectedLegacyAddFieldMap {
			expectedSearchFieldLabels = append(expectedSearchFieldLabels, k)
		}
	}
	originalMap := actualMap.Original()
	actualSearchFieldLabels := make([]search.FieldLabel, 0, len(originalMap))
	for k := range originalMap {
		actualSearchFieldLabels = append(actualSearchFieldLabels, k)
	}
	assert.ElementsMatch(t, expectedSearchFieldLabels, actualSearchFieldLabels)
	for k := range expectedFieldMap {
		field, found := actualMap.Get(k.String())
		assert.Equal(t, expectedFieldMap[k], field)
		assert.True(t, found)
	}
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		for k := range expectedPostgresAddFieldMap {
			field, found := actualMap.Get(k.String())
			assert.Equal(t, expectedPostgresAddFieldMap[k], field)
			assert.True(t, found)
		}
	} else {
		for k := range expectedLegacyAddFieldMap {
			field, found := actualMap.Get(k.String())
			assert.Equal(t, expectedLegacyAddFieldMap[k], field)
			assert.True(t, found)
		}
	}
}
