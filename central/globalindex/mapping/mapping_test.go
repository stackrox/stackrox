package mapping

import (
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
	activeComponentObjContainerNameField = &search.Field{
		FieldPath: getActiveComponentPrefix() + ".active_contexts_slice.container_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_ACTIVE_COMPONENT,
		Analyzer:  "",
	}
	activeComponentObjDeploymentIDField = &search.Field{
		FieldPath: getActiveComponentPrefix() + ".deployment_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_ACTIVE_COMPONENT,
		Analyzer:  "",
	}
	activeComponentObjIDField = &search.Field{
		FieldPath: getActiveComponentPrefix() + ".component_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_ACTIVE_COMPONENT,
		Analyzer:  "",
	}
	activeComponentObjImageIDField = &search.Field{
		FieldPath: getActiveComponentPrefix() + ".active_contexts_slice.image_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_ACTIVE_COMPONENT,
		Analyzer:  "",
	}
	// Field Values - Alert
	alertObjCategoryField = &search.Field{
		FieldPath: getAlertPrefix() + ".policy.categories",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjClusterIDLegacyField = &search.Field{
		FieldPath: getAlertPrefix() + ".common_entity_info.cluster_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjClusterIDPostgresField = &search.Field{
		FieldPath: getAlertPrefix() + ".cluster_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjClusterNameLegacyField = &search.Field{
		FieldPath: getAlertPrefix() + ".common_entity_info.cluster_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjClusterNamePostgresField = &search.Field{
		FieldPath: getAlertPrefix() + ".cluster_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjDeploymentIDField = &search.Field{
		FieldPath: getAlertPrefix() + ".Entity.deployment.id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjDeploymentNameField = &search.Field{
		FieldPath: getAlertPrefix() + ".Entity.deployment.name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjEnforcementLegacyField = &search.Field{
		FieldPath: getAlertPrefix() + ".enforcement_action",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjEnforcementPostgresField = &search.Field{
		FieldPath: getAlertPrefix() + ".enforcement.action",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjInactiveField = &search.Field{
		FieldPath: getAlertPrefix() + ".Entity.deployment.inactive",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjLifecycleStageLegacyField = &search.Field{
		FieldPath: getAlertPrefix() + ".lifecycle_stage",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjLifecycleStagePostgresField = &search.Field{
		FieldPath: getAlertPrefix() + ".lifecycle_stage",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjNamespaceIDLegacyField = &search.Field{
		FieldPath: getAlertPrefix() + ".common_entity_info.namespace_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjNamespaceIDPostgresField = &search.Field{
		FieldPath: getAlertPrefix() + ".namespace_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjNamespaceNameLegacyField = &search.Field{
		FieldPath: getAlertPrefix() + ".common_entity_info.namespace",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjNamespaceNamePostgresField = &search.Field{
		FieldPath: getAlertPrefix() + ".namespace",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjPolicyIDField = &search.Field{
		FieldPath: getAlertPrefix() + ".policy.id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjPolicyNameField = &search.Field{
		FieldPath: getAlertPrefix() + ".policy.name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjPolicySeverityField = &search.Field{
		FieldPath: getAlertPrefix() + ".policy.severity",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjResourceNameField = &search.Field{
		FieldPath: getAlertPrefix() + ".Entity.resource.name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjResourceTypeLegacyField = &search.Field{
		FieldPath: getAlertPrefix() + ".common_entity_info.resource_type",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjResourceTypePostgresField = &search.Field{
		FieldPath: getAlertPrefix() + ".Entity.resource.resource_type",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjSortPolicyNameLegacyField = &search.Field{
		FieldPath: getAlertPrefix() + ".policy.developer_internal_fields.SORT_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "keyword",
	}
	alertObjSortPolicyNamePostgresField = &search.Field{
		FieldPath: getAlertPrefix() + ".policy.SORT_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "keyword",
	}
	alertObjStateField = &search.Field{
		FieldPath: getAlertPrefix() + ".state",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	alertObjViolationTimeField = &search.Field{
		FieldPath: getAlertPrefix() + ".time.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_ALERTS,
		Analyzer:  "",
	}
	// Field Values - Cluster
	clusterObjAdmissionControlStatusField = &search.Field{
		FieldPath: getClusterPrefix() + ".health_status.admission_control_health_status",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	clusterObjClusterStatusField = &search.Field{
		FieldPath: getClusterPrefix() + ".health_status.overall_health_status",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	clusterObjCollectorStatusField = &search.Field{
		FieldPath: getClusterPrefix() + ".health_status.collector_health_status",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	clusterObjIDField = &search.Field{
		FieldPath: getClusterPrefix() + ".id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	clusterObjLabelsField = &search.Field{
		FieldPath: getClusterPrefix() + ".labels",
		Type:      v1.SearchDataType_SEARCH_MAP,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	clusterObjLastContactField = &search.Field{
		FieldPath: getClusterPrefix() + ".health_status.last_contact.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	clusterObjNameField = &search.Field{
		FieldPath: getClusterPrefix() + ".name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	clusterObjScannerStatusField = &search.Field{
		FieldPath: getClusterPrefix() + ".health_status.scanner_health_status",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	clusterObjSensorStatusField = &search.Field{
		FieldPath: getClusterPrefix() + ".health_status.sensor_health_status",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTERS,
		Analyzer:  "",
	}
	// Field Values - ClusterCVE
	clusterCVEObjCVEField = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".cve_base_info.cve",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	clusterCVEObjCVSSField = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".cvss",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	clusterCVEObjCreatedTimeField = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".cve_base_info.created_at.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	clusterCVEObjIDField = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	clusterCVEObjImpactScore = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".impact_score",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	clusterCVEObjPublishedOnField = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".cve_base_info.published_on.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	ClusterCVEObjSeverityField = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".severity",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	clusterCVEObjSnoozedField = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".snoozed",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	clusterCVEObjSnoozeExpiryField = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".snooze_expiry.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	clusterCVEObjTypeField = &search.Field{
		FieldPath: getClusterCVEPrefix() + ".type",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_CLUSTER_VULNERABILITIES,
		Analyzer:  "",
	}
	// Field Values - ClusterCVEEdge
	clusterCVEEdgeObjFixableField = &search.Field{
		FieldPath: getClusterCVEEdgePrefix() + ".is_fixable",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_CLUSTER_VULN_EDGE,
		Analyzer:  "",
	}
	clusterCVEEdgeObjFixedByField = &search.Field{
		FieldPath: getClusterCVEEdgePrefix() + ".HasFixedBy.fixed_by",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_CLUSTER_VULN_EDGE,
		Analyzer:  "",
	}
	// Field Values - ComplianceControl
	complianceControlObjGroupIDField = &search.Field{
		FieldPath: "control.group_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_COMPLIANCE_CONTROL,
		Analyzer:  "",
	}
	complianceControlObjIDField = &search.Field{
		FieldPath: "control.id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_COMPLIANCE_CONTROL,
		Analyzer:  "",
	}
	complianceControlObjNameField = &search.Field{
		FieldPath: "control.name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_COMPLIANCE_CONTROL,
		Analyzer:  "",
	}
	complianceControlObjStandardIDField = &search.Field{
		FieldPath: "control.standard_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_COMPLIANCE_CONTROL,
		Analyzer:  "",
	}
	// Field Values - ComplianceStandard
	complianceStandardObjIDField = &search.Field{
		FieldPath: "standard.metadata.id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_COMPLIANCE_STANDARD,
		Analyzer:  "",
	}
	complianceStandardObjNameField = &search.Field{
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
	// Field Values - Deployment
	deploymentObjAddCapabilitiesField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.security_context.add_capabilities",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjAnnotationsField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".annotations",
		Type:      v1.SearchDataType_SEARCH_MAP,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjClusterIDField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".cluster_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjClusterNameField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".cluster_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjCPUCoresLimitField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.resources.cpu_cores_limit",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjCPUCoresRequestField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.resources.cpu_cores_request",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjCreatedField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".created.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjDropCapabilitiesField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.security_context.drop_capabilities",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjEnvKeyField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.config.env.key",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjEnvValueField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.config.env.value",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjEnvVarSourceField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.config.env.env_var_source",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjExposedNodePortField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.node_port",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjExposingServiceField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.service_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjExposingServicePortField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.service_port",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjExposureLevelField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.level",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjExternalHostnameField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.external_hostnames",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjExternalIPField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".ports.exposure_infos.external_ips",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjIDField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjImageIDField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.image.id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjImageNameField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.image.name.full_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "standard",
	}
	deploymentObjImagePullSecretsField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".image_pull_secrets",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjImageRegistryField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.image.name.registry",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjImageRemoteField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.image.name.remote",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjImageTagField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.image.name.tag",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjLabelsField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".labels",
		Type:      v1.SearchDataType_SEARCH_MAP,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjMaxExposureField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".ports.exposure",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjMemoryLimitField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.resources.memory_mb_limit",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjMemoryRequestField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.resources.memory_mb_request",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjNameField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjNamespaceField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".namespace",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjNamespaceIDField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".namespace_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjOrchestratorComponentField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".orchestrator_component",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjPodLabelField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".pod_labels",
		Type:      v1.SearchDataType_SEARCH_MAP,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjPortField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".ports.container_port",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjPortProtocolField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".ports.protocol",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjPriorityField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".priority",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjPrivilegedField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.security_context.privileged",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjReadOnlyRootFilesystemField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.security_context.read_only_root_filesystem",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjRiskScoreField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".risk_score",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjSecretNameField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.secrets.name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjSecretPathField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.secrets.path",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjServiceAccountNameField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".service_account",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjServiceAccountPermissionLevelField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".service_account_permission_level",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjTypeField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".type",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjVolumeDestinationField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.volumes.destination",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjVolumeNameField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.volumes.name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjVolumeReadOnlyField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.volumes.read_only",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjVolumeSourceField = &search.Field{
		FieldPath: getDeploymentPrefix() + ".containers.volumes.source",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_DEPLOYMENTS,
		Analyzer:  "",
	}
	deploymentObjVolumeTypeField = &search.Field{
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
	imageComponentCVEEdgeObjFixableField = &search.Field{
		FieldPath: getComponentVulnEdgePrefix() + ".is_fixable",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     true,
		Hidden:    false,
		Category:  v1.SearchCategory_COMPONENT_VULN_EDGE,
		Analyzer:  "",
	}
	imageComponentCVEEdgeObjFixedByField = &search.Field{
		FieldPath: getComponentVulnEdgePrefix() + ".HasFixedBy.fixed_by",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_COMPONENT_VULN_EDGE,
		Analyzer:  "",
	}
	// Field Values - ImageComponentEdge
	imageComponentEdgeObjLocationField = &search.Field{
		FieldPath: getImageComponentEdgePrefix() + ".location",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_IMAGE_COMPONENT_EDGE,
		Analyzer:  "",
	}
	// Field Values - ImageCVE
	imageCVEObjCreatedTimeField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".cve_base_info.created_at.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	imageCVEObjCVEField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".cve_base_info.cve",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	imageCVEObjCVSSField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".cvss",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	imageCVEObjIDField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	imageCVEObjImpactScoreField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".impact_score",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    false,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	imageCVEObjOperatingSystemField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".operating_system",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	imageCVEObjPublishedOnField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".cve_base_info.published_on.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	imageCVEObjSeverityField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".severity",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	imageCVEObjSnoozedField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".snoozed",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     false,
		Hidden:    false,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	imageCVEObjSnoozeExpiryField = &search.Field{
		FieldPath: getImageCVEPrefix() + ".snooze_expiry.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    true,
		Category:  getImageCVESearchCategory(),
		Analyzer:  "",
	}
	// Field Values - ImageCVEEdge
	imageCVEEdgeObjFirstOccurrenceField = &search.Field{
		FieldPath: getImageCVEEdgePrefix() + ".first_image_occurrence.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    true,
		Category:  v1.SearchCategory_IMAGE_VULN_EDGE,
		Analyzer:  "",
	}
	imageCVEEdgeObjStateField = &search.Field{
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
	namespaceObjAnnotationsField = &search.Field{
		FieldPath: getNamespacePrefix() + ".annotations",
		Type:      v1.SearchDataType_SEARCH_MAP,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_NAMESPACES,
		Analyzer:  "",
	}
	namespaceObjClusterField = &search.Field{
		FieldPath: getNamespacePrefix() + ".cluster_name",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_NAMESPACES,
		Analyzer:  "",
	}
	namespaceObjClusterIDField = &search.Field{
		FieldPath: getNamespacePrefix() + ".cluster_id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_NAMESPACES,
		Analyzer:  "",
	}
	namespaceObjIDField = &search.Field{
		FieldPath: getNamespacePrefix() + ".id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_NAMESPACES,
		Analyzer:  "",
	}
	namespaceObjLabelField = &search.Field{
		FieldPath: getNamespacePrefix() + ".labels",
		Type:      v1.SearchDataType_SEARCH_MAP,
		Store:     false,
		Hidden:    false,
		Category:  v1.SearchCategory_NAMESPACES,
		Analyzer:  "",
	}
	namespaceObjNameField = &search.Field{
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
	nodeComponentCVEEdgeObjFixableField = &search.Field{
		FieldPath: getNodeComponentVulnEdgePrefix() + ".is_fixable",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     true,
		Hidden:    false,
		Category:  getNodeComponentCVEEdgeSearchCategory(),
		Analyzer:  "",
	}
	nodeComponentCVEEdgeObjFixedByField = &search.Field{
		FieldPath: getNodeComponentVulnEdgePrefix() + ".HasFixedBy.fixed_by",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  getNodeComponentCVEEdgeSearchCategory(),
		Analyzer:  "",
	}
	// Field Values - NodeComponentEdge
	nodeComponentEdgeObjLocationField = &search.Field{
		FieldPath: getNodeComponentEdgePrefix() + ".location",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    true,
		Category:  v1.SearchCategory_NODE_COMPONENT_EDGE,
		Analyzer:  "",
	}
	// Field Values - NodeCVE
	nodeCVEObjCreatedTimeField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".cve_base_info.created_at.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	nodeCVEObjCVEField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".cve_base_info.cve",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     true,
		Hidden:    false,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	nodeCVEObjCVSSField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".cvss",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     true,
		Hidden:    false,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	nodeCVEObjIDField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".id",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    true,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	nodeCVEObjImpactScoreField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".impact_score",
		Type:      v1.SearchDataType_SEARCH_NUMERIC,
		Store:     false,
		Hidden:    false,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	nodeCVEObjOperatingSystemField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".operating_system",
		Type:      v1.SearchDataType_SEARCH_STRING,
		Store:     false,
		Hidden:    false,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	nodeCVEObjPublishedOnField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".cve_base_info.published_on.seconds",
		Type:      v1.SearchDataType_SEARCH_DATETIME,
		Store:     false,
		Hidden:    false,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	nodeCVEObjSeverityField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".severity",
		Type:      v1.SearchDataType_SEARCH_ENUM,
		Store:     false,
		Hidden:    false,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	nodeCVEObjSnoozedField = &search.Field{
		FieldPath: getNodeCVEPrefix() + ".snoozed",
		Type:      v1.SearchDataType_SEARCH_BOOL,
		Store:     false,
		Hidden:    false,
		Category:  getNodeCVESearchCategory(),
		Analyzer:  "",
	}
	nodeCVEObjSnoozeExpiryField = &search.Field{
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
		search.AddCapabilities:               deploymentObjAddCapabilitiesField,
		search.Cluster:                       deploymentObjClusterNameField,
		search.ClusterID:                     deploymentObjClusterIDField,
		search.Component:                     imageObjComponentNameField,
		search.ComponentCount:                imageObjComponentCountField,
		search.ComponentID:                   imageComponentObjIDField,
		search.ComponentLocation:             imageComponentEdgeObjLocationField,
		search.ComponentPriority:             imageComponentObjPriorityField,
		search.ComponentRiskScore:            imageObjComponentRiskScoreField,
		search.ComponentSource:               imageComponentObjSourceField,
		search.ComponentTopCVSS:              imageComponentObjTopCVSSField,
		search.ComponentVersion:              imageObjComponentVersionField,
		search.CPUCoresLimit:                 deploymentObjCPUCoresLimitField,
		search.CPUCoresRequest:               deploymentObjCPUCoresRequestField,
		search.Created:                       deploymentObjCreatedField,
		search.CVE:                           imageObjCVEField,
		search.CVECount:                      imageObjCVECountField,
		search.CVECreatedTime:                imageCVEObjCreatedTimeField,
		search.CVEID:                         imageCVEObjIDField,
		search.CVEPublishedOn:                imageObjCVEPublishedOnField,
		search.CVESuppressed:                 imageObjCVESuppressedField,
		search.CVESuppressExpiry:             imageCVEObjSnoozeExpiryField,
		search.CVSS:                          imageObjCVSSField,
		search.DeploymentAnnotation:          deploymentObjAnnotationsField,
		search.DeploymentID:                  deploymentObjIDField,
		search.DeploymentLabel:               deploymentObjLabelsField,
		search.DeploymentName:                deploymentObjNameField,
		search.DeploymentPriority:            deploymentObjPriorityField,
		search.DeploymentRiskScore:           deploymentObjRiskScoreField,
		search.DeploymentType:                deploymentObjTypeField,
		search.DockerfileInstructionKeyword:  imageObjDockerfileInstructionField,
		search.DockerfileInstructionValue:    imageObjDockerfileInstructionValueField,
		search.DropCapabilities:              deploymentObjDropCapabilitiesField,
		search.EnvironmentKey:                deploymentObjEnvKeyField,
		search.EnvironmentValue:              deploymentObjEnvValueField,
		search.EnvironmentVarSrc:             deploymentObjEnvVarSourceField,
		search.ExposedNodePort:               deploymentObjExposedNodePortField,
		search.ExposingService:               deploymentObjExposingServiceField,
		search.ExposingServicePort:           deploymentObjExposingServicePortField,
		search.ExposureLevel:                 deploymentObjExposureLevelField,
		search.ExternalHostname:              deploymentObjExternalHostnameField,
		search.ExternalIP:                    deploymentObjExternalIPField,
		search.Fixable:                       imageComponentCVEEdgeObjFixableField,
		search.FixableCVECount:               imageObjFixableCVEsField,
		search.FixedBy:                       imageObjFixedByField,
		search.ImageCommand:                  imageObjCommandField,
		search.ImageCreatedTime:              imageObjCreatedTimeField,
		search.ImageEntrypoint:               imageObjEntrypointField,
		search.ImageLabel:                    imageObjLabelField,
		search.ImageName:                     imageObjNameField,
		search.ImageOS:                       imageObjOperatingSystemField,
		search.ImagePriority:                 imageObjPriorityField,
		search.ImagePullSecret:               deploymentObjImagePullSecretsField,
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
		search.ImpactScore:                   imageCVEObjImpactScoreField,
		search.LastUpdatedTime:               imageObjLastUpdatedField,
		search.MaxExposureLevel:              deploymentObjMaxExposureField,
		search.MemoryLimit:                   deploymentObjMemoryLimitField,
		search.MemoryRequest:                 deploymentObjMemoryRequestField,
		search.Namespace:                     deploymentObjNamespaceField,
		search.NamespaceID:                   deploymentObjNamespaceIDField,
		search.OperatingSystem:               imageComponentObjOperatingSystemField,
		search.OrchestratorComponent:         deploymentObjOrchestratorComponentField,
		search.PodLabel:                      deploymentObjPodLabelField,
		search.Port:                          deploymentObjPortField,
		search.PortProtocol:                  deploymentObjPortProtocolField,
		search.Privileged:                    deploymentObjPrivilegedField,
		search.ReadOnlyRootFilesystem:        deploymentObjReadOnlyRootFilesystemField,
		search.SecretName:                    deploymentObjSecretNameField,
		search.SecretPath:                    deploymentObjSecretPathField,
		search.ServiceAccountName:            deploymentObjServiceAccountNameField,
		search.ServiceAccountPermissionLevel: deploymentObjServiceAccountPermissionLevelField,
		search.Severity:                      imageCVEObjSeverityField,
		search.VolumeDestination:             deploymentObjVolumeDestinationField,
		search.VolumeName:                    deploymentObjVolumeNameField,
		search.VolumeReadonly:                deploymentObjVolumeReadOnlyField,
		search.VolumeSource:                  deploymentObjVolumeSourceField,
		search.VolumeType:                    deploymentObjVolumeTypeField,
		search.VulnerabilityState:            imageObjCVEStateField,
	}
	postgresNodeToVulnFieldMap = map[search.FieldLabel]*search.Field{
		search.AdmissionControlStatus: clusterObjAdmissionControlStatusField,
		search.Cluster:                nodeObjClusterNameField,
		search.ClusterID:              nodeObjClusterIDField,
		search.ClusterLabel:           clusterObjLabelsField,
		search.ClusterStatus:          clusterObjClusterStatusField,
		search.CollectorStatus:        clusterObjCollectorStatusField,
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
		search.CVEID:                  nodeCVEObjIDField,
		search.CVEPublishedOn:         nodeObjCVEPublishedOnField,
		search.CVESuppressed:          nodeObjCVESnoozedField,
		search.CVESuppressExpiry:      nodeCVEObjSnoozeExpiryField,
		search.CVSS:                   nodeObjCVSSField,
		search.Fixable:                nodeComponentCVEEdgeObjFixableField,
		search.FixableCVECount:        nodeObjFixableCVECountField,
		search.FixedBy:                nodeObjFixedByField,
		search.ImpactScore:            nodeCVEObjImpactScoreField,
		search.LastContactTime:        clusterObjLastContactField,
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
		search.ScannerStatus:          clusterObjScannerStatusField,
		search.SensorStatus:           clusterObjSensorStatusField,
		search.Severity:               nodeCVEObjSeverityField,
		search.TaintKey:               nodeObjTaintKeyField,
		search.TaintValue:             nodeObjTaintValueField,
		search.TolerationEffect:       nodeObjTaintEffectField,
		search.VulnerabilityState:     nodeObjVulnerabilityStateField,
	}
)

func TestActiveComponentMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.ComponentID:   activeComponentObjIDField,
		search.ContainerName: activeComponentObjContainerNameField,
		search.DeploymentID:  activeComponentObjDeploymentIDField,
		search.ImageSHA:      activeComponentObjImageIDField,
	}
	validateOptionsMap(t,
		v1.SearchCategory_ACTIVE_COMPONENT,
		expectedSearchFieldMap,
		nil,
		nil)
}

func TestAlertMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Category:       alertObjCategoryField,
		search.DeploymentID:   alertObjDeploymentIDField,
		search.DeploymentName: alertObjDeploymentNameField,
		search.Inactive:       alertObjInactiveField,
		search.PolicyID:       alertObjPolicyIDField,
		search.PolicyName:     alertObjPolicyNameField,
		search.ResourceName:   alertObjResourceNameField,
		search.Severity:       alertObjPolicySeverityField,
		search.ViolationState: alertObjStateField,
		search.ViolationTime:  alertObjViolationTimeField,
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Cluster:        alertObjClusterNameLegacyField,
		search.ClusterID:      alertObjClusterIDLegacyField,
		search.Enforcement:    alertObjEnforcementLegacyField,
		search.LifecycleStage: alertObjLifecycleStageLegacyField,
		search.Namespace:      alertObjNamespaceNameLegacyField,
		search.NamespaceID:    alertObjNamespaceIDLegacyField,
		search.ResourceType:   alertObjResourceTypeLegacyField,
		search.SORTPolicyName: alertObjSortPolicyNameLegacyField,
	}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.Cluster:        alertObjClusterNamePostgresField,
		search.ClusterID:      alertObjClusterIDPostgresField,
		search.Enforcement:    alertObjEnforcementPostgresField,
		search.LifecycleStage: alertObjLifecycleStagePostgresField,
		search.Namespace:      alertObjNamespaceNamePostgresField,
		search.NamespaceID:    alertObjNamespaceIDPostgresField,
		search.ResourceType:   alertObjResourceTypePostgresField,
		search.SORTPolicyName: alertObjSortPolicyNamePostgresField,
	}
	validateOptionsMap(t,
		v1.SearchCategory_ALERTS,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestClusterVulnEdgeMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.ClusterCVEFixable: clusterCVEEdgeObjFixableField,
		search.ClusterCVEFixedBy: clusterCVEEdgeObjFixedByField,
	}
	expectedLegacySearchFieldMap := map[search.FieldLabel]*search.Field{}
	expectedPostgresSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.AdmissionControlStatus: clusterObjAdmissionControlStatusField,
		search.Cluster:                clusterObjNameField,
		search.ClusterID:              clusterObjIDField,
		search.ClusterLabel:           clusterObjLabelsField,
		search.ClusterStatus:          clusterObjClusterStatusField,
		search.CollectorStatus:        clusterObjCollectorStatusField,
		search.CVE:                    clusterCVEObjCVEField,
		search.CVECreatedTime:         clusterCVEObjCreatedTimeField,
		search.CVEID:                  clusterCVEObjIDField,
		search.CVEPublishedOn:         clusterCVEObjPublishedOnField,
		search.CVESuppressed:          clusterCVEObjSnoozedField,
		search.CVESuppressExpiry:      clusterCVEObjSnoozeExpiryField,
		search.CVEType:                clusterCVEObjTypeField,
		search.CVSS:                   clusterCVEObjCVSSField,
		search.ImpactScore:            clusterCVEObjImpactScore,
		search.LastContactTime:        clusterObjLastContactField,
		search.ScannerStatus:          clusterObjScannerStatusField,
		search.SensorStatus:           clusterObjSensorStatusField,
		search.Severity:               ClusterCVEObjSeverityField,
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
		search.AdmissionControlStatus: clusterObjAdmissionControlStatusField,
		search.Cluster:                clusterObjNameField,
		search.ClusterCVEFixable:      clusterCVEEdgeObjFixableField,
		search.ClusterCVEFixedBy:      clusterCVEEdgeObjFixedByField,
		search.ClusterID:              clusterObjIDField,
		search.ClusterLabel:           clusterObjLabelsField,
		search.ClusterStatus:          clusterObjClusterStatusField,
		search.CollectorStatus:        clusterObjCollectorStatusField,
		search.CVE:                    clusterCVEObjCVEField,
		search.CVECreatedTime:         clusterCVEObjCreatedTimeField,
		search.CVEID:                  clusterCVEObjIDField,
		search.CVEPublishedOn:         clusterCVEObjPublishedOnField,
		search.CVESuppressed:          clusterCVEObjSnoozedField,
		search.CVESuppressExpiry:      clusterCVEObjSnoozeExpiryField,
		search.CVEType:                clusterCVEObjTypeField,
		search.CVSS:                   clusterCVEObjCVSSField,
		search.ImpactScore:            clusterCVEObjImpactScore,
		search.LastContactTime:        clusterObjLastContactField,
		search.ScannerStatus:          clusterObjScannerStatusField,
		search.SensorStatus:           clusterObjSensorStatusField,
		search.Severity:               ClusterCVEObjSeverityField,
	}
	validateOptionsMap(t,
		v1.SearchCategory_CLUSTER_VULNERABILITIES,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestClustersMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.AdmissionControlStatus: clusterObjAdmissionControlStatusField,
		search.Cluster:                clusterObjNameField,
		search.ClusterID:              clusterObjIDField,
		search.ClusterLabel:           clusterObjLabelsField,
		search.ClusterStatus:          clusterObjClusterStatusField,
		search.CollectorStatus:        clusterObjCollectorStatusField,
		search.LastContactTime:        clusterObjLastContactField,
		search.ScannerStatus:          clusterObjScannerStatusField,
		search.SensorStatus:           clusterObjSensorStatusField,
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
		search.Control:        complianceControlObjNameField,
		search.ControlID:      complianceControlObjIDField,
		search.ControlGroupID: complianceControlObjGroupIDField,
		search.StandardID:     complianceControlObjStandardIDField,
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
		search.Standard:   complianceStandardObjNameField,
		search.StandardID: complianceStandardObjIDField,
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
		search.Fixable: imageComponentCVEEdgeObjFixableField,
		search.FixedBy: imageComponentCVEEdgeObjFixedByField,
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
		search.AddCapabilities:               deploymentObjAddCapabilitiesField,
		search.Cluster:                       deploymentObjClusterNameField,
		search.ClusterID:                     deploymentObjClusterIDField,
		search.Component:                     imageObjComponentNameField,
		search.ComponentCount:                imageObjComponentCountField,
		search.ComponentRiskScore:            imageObjComponentRiskScoreField,
		search.ComponentVersion:              imageObjComponentVersionField,
		search.ContainerID:                   processIndicatorObjContainerIDField,
		search.ContainerName:                 processIndicatorObjContainerNameField,
		search.CPUCoresLimit:                 deploymentObjCPUCoresLimitField,
		search.CPUCoresRequest:               deploymentObjCPUCoresRequestField,
		search.Created:                       deploymentObjCreatedField,
		search.CVE:                           imageObjCVEField,
		search.CVECount:                      imageObjCVECountField,
		search.CVEPublishedOn:                imageObjCVEPublishedOnField,
		search.CVESuppressed:                 imageObjCVESuppressedField,
		search.CVSS:                          imageObjCVSSField,
		search.DeploymentAnnotation:          deploymentObjAnnotationsField,
		search.DeploymentID:                  deploymentObjIDField,
		search.DeploymentLabel:               deploymentObjLabelsField,
		search.DeploymentName:                deploymentObjNameField,
		search.DeploymentPriority:            deploymentObjPriorityField,
		search.DeploymentRiskScore:           deploymentObjRiskScoreField,
		search.DeploymentType:                deploymentObjTypeField,
		search.DockerfileInstructionKeyword:  imageObjDockerfileInstructionField,
		search.DockerfileInstructionValue:    imageObjDockerfileInstructionValueField,
		search.DropCapabilities:              deploymentObjDropCapabilitiesField,
		search.EnvironmentKey:                deploymentObjEnvKeyField,
		search.EnvironmentValue:              deploymentObjEnvValueField,
		search.EnvironmentVarSrc:             deploymentObjEnvVarSourceField,
		search.ExposedNodePort:               deploymentObjExposedNodePortField,
		search.ExposingService:               deploymentObjExposingServiceField,
		search.ExposingServicePort:           deploymentObjExposingServicePortField,
		search.ExposureLevel:                 deploymentObjExposureLevelField,
		search.ExternalHostname:              deploymentObjExternalHostnameField,
		search.ExternalIP:                    deploymentObjExternalIPField,
		search.FixableCVECount:               imageObjFixableCVEsField,
		search.FixedBy:                       imageObjFixedByField,
		search.ImageCommand:                  imageObjCommandField,
		search.ImageCreatedTime:              imageObjCreatedTimeField,
		search.ImageEntrypoint:               imageObjEntrypointField,
		search.ImageLabel:                    imageObjLabelField,
		search.ImageName:                     deploymentObjImageNameField,
		search.ImageOS:                       imageObjOperatingSystemField,
		search.ImagePriority:                 imageObjPriorityField,
		search.ImagePullSecret:               deploymentObjImagePullSecretsField,
		search.ImageRegistry:                 deploymentObjImageRegistryField,
		search.ImageRemote:                   deploymentObjImageRemoteField,
		search.ImageRiskScore:                imageObjRiskScoreField,
		search.ImageScanTime:                 imageObjScanTimeField,
		search.ImageSHA:                      deploymentObjImageIDField,
		search.ImageSignatureFetchedTime:     imageObjSignatureFetchTimeField,
		search.ImageTag:                      deploymentObjImageTagField,
		search.ImageTopCVSS:                  imageObjTopCVSSField,
		search.ImageUser:                     imageObjUserField,
		search.ImageVolumes:                  imageObjVolumesField,
		search.LastUpdatedTime:               imageObjLastUpdatedField,
		search.MaxExposureLevel:              deploymentObjMaxExposureField,
		search.MemoryLimit:                   deploymentObjMemoryLimitField,
		search.MemoryRequest:                 deploymentObjMemoryRequestField,
		search.Namespace:                     deploymentObjNamespaceField,
		search.NamespaceID:                   deploymentObjNamespaceIDField,
		search.OrchestratorComponent:         deploymentObjOrchestratorComponentField,
		search.PodID:                         processIndicatorObjPodIDField,
		search.PodLabel:                      processIndicatorObjPodLabelField,
		search.PodUID:                        processIndicatorObjPodUIDField,
		search.Port:                          deploymentObjPortField,
		search.PortProtocol:                  deploymentObjPortProtocolField,
		search.Privileged:                    deploymentObjPrivilegedField,
		search.ProcessArguments:              processIndicatorObjProcessArgumentsField,
		search.ProcessExecPath:               processIndicatorObjProcessExecPathField,
		search.ProcessID:                     processIndicatorObjProcessIDField,
		search.ProcessName:                   processIndicatorObjProcessNameField,
		search.ProcessUID:                    processIndicatorObjProcessUIDField,
		search.ReadOnlyRootFilesystem:        deploymentObjReadOnlyRootFilesystemField,
		search.SecretName:                    deploymentObjSecretNameField,
		search.SecretPath:                    deploymentObjSecretPathField,
		search.ServiceAccountName:            deploymentObjServiceAccountNameField,
		search.ServiceAccountPermissionLevel: deploymentObjServiceAccountPermissionLevelField,
		search.VolumeDestination:             deploymentObjVolumeDestinationField,
		search.VolumeName:                    deploymentObjVolumeNameField,
		search.VolumeReadonly:                deploymentObjVolumeReadOnlyField,
		search.VolumeSource:                  deploymentObjVolumeSourceField,
		search.VolumeType:                    deploymentObjVolumeTypeField,
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
		search.Cluster:                      deploymentObjClusterNameField,
		search.ClusterID:                    deploymentObjClusterIDField,
		search.Component:                    imageObjComponentNameField,
		search.ComponentCount:               imageObjComponentCountField,
		search.ComponentID:                  imageComponentObjIDField,
		search.ComponentLocation:            imageComponentEdgeObjLocationField,
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
		search.DeploymentID:                 deploymentObjIDField,
		search.DeploymentLabel:              deploymentObjLabelsField,
		search.DeploymentName:               deploymentObjNameField,
		search.DockerfileInstructionKeyword: imageObjDockerfileInstructionField,
		search.DockerfileInstructionValue:   imageObjDockerfileInstructionValueField,
		search.Fixable:                      imageComponentCVEEdgeObjFixableField,
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
		search.Namespace:                    deploymentObjNamespaceField,
		search.NamespaceID:                  deploymentObjNamespaceIDField,
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
		search.Cluster:                      deploymentObjClusterNameField,
		search.ClusterID:                    deploymentObjClusterIDField,
		search.Component:                    imageComponentObjNameField,
		search.ComponentCount:               imageObjComponentCountField,
		search.ComponentID:                  imageComponentObjIDField,
		search.ComponentPriority:            imageComponentObjPriorityField,
		search.ComponentLocation:            imageComponentEdgeObjLocationField,
		search.ComponentRiskScore:           imageComponentObjRiskScoreField,
		search.ComponentSource:              imageComponentObjSourceField,
		search.ComponentTopCVSS:             imageComponentObjTopCVSSField,
		search.ComponentVersion:             imageComponentObjVersionField,
		search.CVE:                          imageObjCVEField,
		search.CVECount:                     imageObjCVECountField,
		search.CVEPublishedOn:               imageObjCVEPublishedOnField,
		search.CVESuppressed:                imageObjCVESuppressedField,
		search.CVSS:                         imageObjCVSSField,
		search.DeploymentID:                 deploymentObjIDField,
		search.DeploymentLabel:              deploymentObjLabelsField,
		search.DeploymentName:               deploymentObjNameField,
		search.DockerfileInstructionKeyword: imageObjDockerfileInstructionField,
		search.DockerfileInstructionValue:   imageObjDockerfileInstructionValueField,
		search.Fixable:                      imageComponentCVEEdgeObjFixableField,
		search.FixableCVECount:              imageObjFixableCVEsField,
		search.FixedBy:                      imageComponentCVEEdgeObjFixedByField,
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
		search.ImpactScore:                  imageCVEObjImpactScoreField,
		search.LastUpdatedTime:              imageObjLastUpdatedField,
		search.Namespace:                    deploymentObjNamespaceField,
		search.NamespaceID:                  deploymentObjNamespaceIDField,
		search.OperatingSystem:              imageComponentObjOperatingSystemField,
		search.Severity:                     imageCVEObjSeverityField,
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
	expectedPostgresSearchFieldMap[search.CVECreatedTime] = imageCVEObjCreatedTimeField
	expectedPostgresSearchFieldMap[search.CVESuppressExpiry] = imageCVEObjSnoozeExpiryField
	validateOptionsMap(t,
		v1.SearchCategory_IMAGE_COMPONENTS,
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestImageComponentEdgeMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.ComponentLocation: imageComponentEdgeObjLocationField,
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
		search.AddCapabilities:               deploymentObjAddCapabilitiesField,
		search.Cluster:                       deploymentObjClusterNameField,
		search.ClusterID:                     deploymentObjClusterIDField,
		search.Component:                     imageComponentObjNameField,
		search.ComponentCount:                imageObjComponentCountField,
		search.ComponentID:                   imageComponentObjIDField,
		search.ComponentLocation:             imageComponentEdgeObjLocationField,
		search.ComponentPriority:             imageComponentObjPriorityField,
		search.ComponentRiskScore:            imageComponentObjRiskScoreField,
		search.ComponentSource:               imageComponentObjSourceField,
		search.ComponentTopCVSS:              imageComponentObjTopCVSSField,
		search.ComponentVersion:              imageComponentObjVersionField,
		search.ContainerID:                   processIndicatorObjContainerIDField,
		search.ContainerName:                 processIndicatorObjContainerNameField,
		search.CPUCoresLimit:                 deploymentObjCPUCoresLimitField,
		search.CPUCoresRequest:               deploymentObjCPUCoresRequestField,
		search.Created:                       deploymentObjCreatedField,
		search.CVECount:                      imageObjCVECountField,
		search.DeploymentAnnotation:          deploymentObjAnnotationsField,
		search.DeploymentID:                  deploymentObjIDField,
		search.DeploymentLabel:               deploymentObjLabelsField,
		search.DeploymentName:                deploymentObjNameField,
		search.DeploymentPriority:            deploymentObjPriorityField,
		search.DeploymentRiskScore:           deploymentObjRiskScoreField,
		search.DeploymentType:                deploymentObjTypeField,
		search.DockerfileInstructionKeyword:  imageObjDockerfileInstructionField,
		search.DockerfileInstructionValue:    imageObjDockerfileInstructionValueField,
		search.DropCapabilities:              deploymentObjDropCapabilitiesField,
		search.EnvironmentKey:                deploymentObjEnvKeyField,
		search.EnvironmentValue:              deploymentObjEnvValueField,
		search.EnvironmentVarSrc:             deploymentObjEnvVarSourceField,
		search.ExposedNodePort:               deploymentObjExposedNodePortField,
		search.ExposingService:               deploymentObjExposingServiceField,
		search.ExposingServicePort:           deploymentObjExposingServicePortField,
		search.ExposureLevel:                 deploymentObjExposureLevelField,
		search.ExternalHostname:              deploymentObjExternalHostnameField,
		search.ExternalIP:                    deploymentObjExternalIPField,
		search.Fixable:                       imageComponentCVEEdgeObjFixableField,
		search.FixableCVECount:               imageObjFixableCVEsField,
		search.FixedBy:                       imageComponentCVEEdgeObjFixedByField,
		search.ImageCommand:                  imageObjCommandField,
		search.ImageCreatedTime:              imageObjCreatedTimeField,
		search.ImageEntrypoint:               imageObjEntrypointField,
		search.ImageLabel:                    imageObjLabelField,
		search.ImageName:                     imageObjNameField,
		search.ImageOS:                       imageObjOperatingSystemField,
		search.ImagePriority:                 imageObjPriorityField,
		search.ImagePullSecret:               deploymentObjImagePullSecretsField,
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
		search.MaxExposureLevel:              deploymentObjMaxExposureField,
		search.MemoryLimit:                   deploymentObjMemoryLimitField,
		search.MemoryRequest:                 deploymentObjMemoryRequestField,
		search.Namespace:                     deploymentObjNamespaceField,
		search.NamespaceID:                   deploymentObjNamespaceIDField,
		search.OrchestratorComponent:         deploymentObjOrchestratorComponentField,
		search.PodLabel:                      deploymentObjPodLabelField,
		search.PodID:                         processIndicatorObjPodIDField,
		search.PodUID:                        processIndicatorObjPodUIDField,
		search.Port:                          deploymentObjPortField,
		search.PortProtocol:                  deploymentObjPortProtocolField,
		search.Privileged:                    deploymentObjPrivilegedField,
		search.ProcessArguments:              processIndicatorObjProcessArgumentsField,
		search.ProcessExecPath:               processIndicatorObjProcessExecPathField,
		search.ProcessID:                     processIndicatorObjProcessIDField,
		search.ProcessName:                   processIndicatorObjProcessNameField,
		search.ProcessUID:                    processIndicatorObjProcessUIDField,
		search.ReadOnlyRootFilesystem:        deploymentObjReadOnlyRootFilesystemField,
		search.SecretName:                    deploymentObjSecretNameField,
		search.SecretPath:                    deploymentObjSecretPathField,
		search.ServiceAccountName:            deploymentObjServiceAccountNameField,
		search.ServiceAccountPermissionLevel: deploymentObjServiceAccountPermissionLevelField,
		search.VolumeDestination:             deploymentObjVolumeDestinationField,
		search.VolumeName:                    deploymentObjVolumeNameField,
		search.VolumeReadonly:                deploymentObjVolumeReadOnlyField,
		search.VolumeSource:                  deploymentObjVolumeSourceField,
		search.VolumeType:                    deploymentObjVolumeTypeField,
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
		search.AdmissionControlStatus:        clusterObjAdmissionControlStatusField,
		search.ClusterLabel:                  clusterObjLabelsField,
		search.ClusterStatus:                 clusterObjClusterStatusField,
		search.CollectorStatus:               clusterObjCollectorStatusField,
		search.CVE:                           imageCVEObjCVEField,
		search.CVECreatedTime:                imageCVEObjCreatedTimeField,
		search.CVEID:                         imageCVEObjIDField,
		search.CVEPublishedOn:                imageCVEObjPublishedOnField,
		search.CVESuppressed:                 imageCVEObjSnoozedField,
		search.CVESuppressExpiry:             imageCVEObjSnoozeExpiryField,
		search.CVSS:                          imageCVEObjCVSSField,
		search.FirstImageOccurrenceTimestamp: imageCVEEdgeObjFirstOccurrenceField,
		search.ImpactScore:                   imageCVEObjImpactScoreField,
		search.LastContactTime:               clusterObjLastContactField,
		search.NamespaceAnnotation:           namespaceObjAnnotationsField,
		search.NamespaceLabel:                namespaceObjLabelField,
		search.OperatingSystem:               imageCVEObjOperatingSystemField,
		search.ScannerStatus:                 clusterObjScannerStatusField,
		search.SensorStatus:                  clusterObjSensorStatusField,
		search.Severity:                      imageCVEObjSeverityField,
	}
	validateOptionsMap(t,
		getImageCVESearchCategory(),
		expectedSearchFieldMap,
		expectedLegacySearchFieldMap,
		expectedPostgresSearchFieldMap)
}

func TestImageCVEEdgeMapping(t *testing.T) {
	expectedSearchFieldMap := map[search.FieldLabel]*search.Field{
		search.FirstImageOccurrenceTimestamp: imageCVEEdgeObjFirstOccurrenceField,
		search.VulnerabilityState:            imageCVEEdgeObjStateField,
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
		search.Cluster:             namespaceObjClusterField,
		search.ClusterID:           namespaceObjClusterIDField,
		search.Namespace:           namespaceObjNameField,
		search.NamespaceAnnotation: namespaceObjAnnotationsField,
		search.NamespaceID:         namespaceObjIDField,
		search.NamespaceLabel:      namespaceObjLabelField,
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
		search.Fixable:            nodeComponentCVEEdgeObjFixableField,
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
