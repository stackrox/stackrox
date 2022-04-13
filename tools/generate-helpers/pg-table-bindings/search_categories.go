package main

import v1 "github.com/stackrox/rox/generated/api/v1"

var (
	typeToSearchCategoryMap = map[string]v1.SearchCategory{
		"":                              v1.SearchCategory_SEARCH_UNSET,
		"Alert":                         v1.SearchCategory_ALERTS,
		"Image":                         v1.SearchCategory_IMAGES,
		"ImageComponent":                v1.SearchCategory_IMAGE_COMPONENTS,
		"ImageCVEEdge":                  v1.SearchCategory_IMAGE_VULN_EDGE,
		"ImageComponentEdge":            v1.SearchCategory_IMAGE_COMPONENT_EDGE,
		"Policy":                        v1.SearchCategory_POLICIES,
		"Deployment":                    v1.SearchCategory_DEPLOYMENTS,
		"ActiveComponent_ActiveContext": v1.SearchCategory_ACTIVE_COMPONENT,
		"Pod":                           v1.SearchCategory_PODS,
		"Secret":                        v1.SearchCategory_SECRETS,
		"ProcessIndicator":              v1.SearchCategory_PROCESS_INDICATORS,
		// "": v1.SearchCategory_COMPLIANCE ,
		"Cluster":            v1.SearchCategory_CLUSTERS,
		"NamespaceMetadata":  v1.SearchCategory_NAMESPACES,
		"Node":               v1.SearchCategory_NODES,
		"NodeCVEEdge":        v1.SearchCategory_NODE_VULN_EDGE,
		"NodeComponentEdge":  v1.SearchCategory_NODE_COMPONENT_EDGE,
		"ComplianceStandard": v1.SearchCategory_COMPLIANCE_STANDARD,
		// "": v1.SearchCategory_COMPLIANCE_CONTROL_GROUP ,
		"ComplianceControl":    v1.SearchCategory_COMPLIANCE_CONTROL,
		"ServiceAccount":       v1.SearchCategory_SERVICE_ACCOUNTS,
		"Role":                 v1.SearchCategory_ROLES,
		"K8SRoleBinding":       v1.SearchCategory_ROLEBINDINGS,
		"ReportConfiguration":  v1.SearchCategory_REPORT_CONFIGURATIONS,
		"ProcessBaseline":      v1.SearchCategory_PROCESS_BASELINES,
		"Subject":              v1.SearchCategory_SUBJECTS,
		"Risk":                 v1.SearchCategory_RISKS,
		"ImageCVE":             v1.SearchCategory_VULNERABILITIES,
		"ClusterCVE":           v1.SearchCategory_CLUSTER_VULNERABILITIES,
		"NodeCVE":              v1.SearchCategory_NODE_VULNERABILITIES,
		"ComponentCVEEdge":     v1.SearchCategory_COMPONENT_VULN_EDGE,
		"ClusterCVEEdge":       v1.SearchCategory_CLUSTER_VULN_EDGE,
		"NetworkEntity":        v1.SearchCategory_NETWORK_ENTITY,
		"VulnerabilityRequest": v1.SearchCategory_VULN_REQUEST,
	}
)
