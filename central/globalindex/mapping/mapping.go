package mapping

import (
	alertMapping "github.com/stackrox/rox/central/alert/mappings"
	"github.com/stackrox/rox/central/compliance/standards/index"
	subjectMapping "github.com/stackrox/rox/central/rbac/service/mapping"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
)

// GetEntityOptionsMap is a mapping from search categories to the options
func GetEntityOptionsMap() map[v1.SearchCategory]search.OptionsMap {
	// Note: with the dackbox graph split brought with the postgres migration, the concept
	// of CVE was split into ClusterCVE, ImageCVE and NodeCVE. The old content seems to focus
	// mostly on image CVEs.
	// Note: with the dackbox graph split brought with the postgres migration, the concept
	// of Component (ImageComponent) was split into ImageComponent and NodeComponent.
	// The old content seems to focus mostly on image Components.
	clusterToVulnerabilitySearchOptions := search.CombineOptionsMaps(
		schema.ClusterCvesSchema.OptionsMap,
		schema.ClusterCveEdgesSchema.OptionsMap,
		schema.ClustersSchema.OptionsMap,
	)

	deploymentsCustomSearchOptions := search.CombineOptionsMaps(
		schema.DeploymentsSchema.OptionsMap,
		schema.ImagesSchema.OptionsMap,
		schema.ProcessIndicatorsSchema.OptionsMap,
	)

	imageToVulnerabilitySearchOptions := search.CombineOptionsMaps(
		schema.ImageCvesSchema.OptionsMap,
		schema.ImageCveEdgesSchema.OptionsMap,
		schema.ImageComponentCveEdgesSchema.OptionsMap,
		schema.ImageComponentsSchema.OptionsMap,
		schema.ImageComponentEdgesSchema.OptionsMap,
		schema.ImagesSchema.OptionsMap,
		schema.DeploymentsSchema.OptionsMap,
	)

	nodeToVulnerabilitySearchOptions := search.CombineOptionsMaps(
		schema.NodeCvesSchema.OptionsMap,
		schema.NodeComponentsCvesEdgesSchema.OptionsMap,
		schema.NodeComponentsSchema.OptionsMap,
		schema.NodeComponentEdgesSchema.OptionsMap,
		schema.NodesSchema.OptionsMap,
	)

	// alerts has a reconciliation mechanism implemented in postgres mode in order to keep only search fields
	// linked to the ListAlert type (goal is backward compatibility with the search strategy implemented on bleve.
	alertSearchOptions := alertMapping.OptionsMap

	subjectSearchOptions := search.CombineOptionsMaps(
		subjectMapping.OptionsMap,
		schema.RoleBindingsSchema.OptionsMap,
	)

	// EntityOptionsMap is a mapping from search categories to the options map for that category.
	// search document maps are also built off this map
	entityOptionsMap := map[v1.SearchCategory]search.OptionsMap{
		v1.SearchCategory_ACTIVE_COMPONENT:        schema.ActiveComponentsSchema.OptionsMap,
		v1.SearchCategory_ALERTS:                  alertSearchOptions,
		v1.SearchCategory_CLUSTER_VULN_EDGE:       clusterToVulnerabilitySearchOptions,
		v1.SearchCategory_CLUSTER_VULNERABILITIES: clusterToVulnerabilitySearchOptions,
		v1.SearchCategory_CLUSTERS:                schema.ClustersSchema.OptionsMap,
		v1.SearchCategory_COMPLIANCE_STANDARD:     index.StandardOptions,
		v1.SearchCategory_COMPLIANCE_CONTROL:      index.ControlOptions,
		v1.SearchCategory_COMPONENT_VULN_EDGE:     imageToVulnerabilitySearchOptions,
		v1.SearchCategory_DEPLOYMENTS:             deploymentsCustomSearchOptions,
		v1.SearchCategory_IMAGE_COMPONENT_EDGE:    imageToVulnerabilitySearchOptions,
		v1.SearchCategory_IMAGE_COMPONENTS:        imageToVulnerabilitySearchOptions,
		v1.SearchCategory_IMAGE_INTEGRATIONS:      schema.ImageIntegrationsSchema.OptionsMap,
		v1.SearchCategory_IMAGE_VULN_EDGE:         imageToVulnerabilitySearchOptions,
		v1.SearchCategory_IMAGE_VULNERABILITIES:   imageToVulnerabilitySearchOptions,
		v1.SearchCategory_IMAGES:                  imageToVulnerabilitySearchOptions,
		v1.SearchCategory_NAMESPACES:              schema.NamespacesSchema.OptionsMap,
		v1.SearchCategory_NODE_COMPONENT_EDGE:     nodeToVulnerabilitySearchOptions,
		v1.SearchCategory_NODE_COMPONENTS:         nodeToVulnerabilitySearchOptions,
		v1.SearchCategory_NODE_VULNERABILITIES:    nodeToVulnerabilitySearchOptions,
		v1.SearchCategory_NODES:                   nodeToVulnerabilitySearchOptions,
		v1.SearchCategory_PODS:                    schema.PodsSchema.OptionsMap,
		v1.SearchCategory_POLICIES:                schema.PoliciesSchema.OptionsMap,
		v1.SearchCategory_POLICY_CATEGORIES:       schema.PolicyCategoriesSchema.OptionsMap,
		v1.SearchCategory_PROCESS_BASELINES:       schema.ProcessBaselinesSchema.OptionsMap,
		v1.SearchCategory_PROCESS_INDICATORS:      schema.ProcessIndicatorsSchema.OptionsMap,
		v1.SearchCategory_REPORT_CONFIGURATIONS:   schema.ReportConfigurationsSchema.OptionsMap,
		v1.SearchCategory_RISKS:                   schema.RisksSchema.OptionsMap,
		v1.SearchCategory_ROLES:                   schema.K8sRolesSchema.OptionsMap,
		v1.SearchCategory_ROLEBINDINGS:            schema.RoleBindingsSchema.OptionsMap,
		v1.SearchCategory_SECRETS:                 schema.SecretsSchema.OptionsMap,
		v1.SearchCategory_SERVICE_ACCOUNTS:        schema.ServiceAccountsSchema.OptionsMap,
		v1.SearchCategory_SUBJECTS:                subjectSearchOptions,
		v1.SearchCategory_VULN_REQUEST:            schema.VulnerabilityRequestsSchema.OptionsMap,
	}

	if features.VulnReportingEnhancements.Enabled() {
		entityOptionsMap[v1.SearchCategory_REPORT_SNAPSHOT] = schema.ReportSnapshotsSchema.OptionsMap

		reportConfigurationSearchOptions := search.CombineOptionsMaps(
			schema.ReportConfigurationsSchema.OptionsMap,
			schema.ReportSnapshotsSchema.OptionsMap,
		)
		entityOptionsMap[v1.SearchCategory_REPORT_CONFIGURATIONS] = reportConfigurationSearchOptions
	}

	return entityOptionsMap
}
