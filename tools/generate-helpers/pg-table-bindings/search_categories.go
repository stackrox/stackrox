package main

import v1 "github.com/stackrox/rox/generated/api/v1"

var (
	typeToSearchCategoryMap = map[string]v1.SearchCategory{
		"":                              v1.SearchCategory_SEARCH_UNSET,
		"ActiveComponent_ActiveContext": v1.SearchCategory_ACTIVE_COMPONENT,
		"Alert":                         v1.SearchCategory_ALERTS,
		"Cluster":                       v1.SearchCategory_CLUSTERS,
		"ClusterCVE":                    v1.SearchCategory_CLUSTER_VULNERABILITIES,
		"ClusterCVEEdge":                v1.SearchCategory_CLUSTER_VULN_EDGE,
		// "": v1.SearchCategory_COMPLIANCE ,
		"ComplianceControl":  v1.SearchCategory_COMPLIANCE_CONTROL,
		"ComplianceStandard": v1.SearchCategory_COMPLIANCE_STANDARD,
		// "": v1.SearchCategory_COMPLIANCE_CONTROL_GROUP ,
		"ComponentCVEEdge":     v1.SearchCategory_COMPONENT_VULN_EDGE,
		"CVE":                  v1.SearchCategory_VULNERABILITIES,
		"Deployment":           v1.SearchCategory_DEPLOYMENTS,
		"Image":                v1.SearchCategory_IMAGES,
		"ImageComponent":       v1.SearchCategory_IMAGE_COMPONENTS,
		"ImageComponentEdge":   v1.SearchCategory_IMAGE_COMPONENT_EDGE,
		"ImageCVEEdge":         v1.SearchCategory_IMAGE_VULN_EDGE,
		"K8SRole":              v1.SearchCategory_ROLES,
		"K8SRoleBinding":       v1.SearchCategory_ROLEBINDINGS,
		"NamespaceMetadata":    v1.SearchCategory_NAMESPACES,
		"NetworkEntity":        v1.SearchCategory_NETWORK_ENTITY,
		"Node":                 v1.SearchCategory_NODES,
		"NodeComponentEdge":    v1.SearchCategory_NODE_COMPONENT_EDGE,
		"NodeCVE":              v1.SearchCategory_NODE_VULNERABILITIES,
		"NodeCVEEdge":          v1.SearchCategory_NODE_VULN_EDGE,
		"Pod":                  v1.SearchCategory_PODS,
		"Policy":               v1.SearchCategory_POLICIES,
		"ProcessBaseline":      v1.SearchCategory_PROCESS_BASELINES,
		"ProcessIndicator":     v1.SearchCategory_PROCESS_INDICATORS,
		"ReportConfiguration":  v1.SearchCategory_REPORT_CONFIGURATIONS,
		"Risk":                 v1.SearchCategory_RISKS,
		"Secret":               v1.SearchCategory_SECRETS,
		"ServiceAccount":       v1.SearchCategory_SERVICE_ACCOUNTS,
		"Subject":              v1.SearchCategory_SUBJECTS,
		"VulnerabilityRequest": v1.SearchCategory_VULN_REQUEST,

		"TestMultiKeyStruct":  v1.SearchCategory_SEARCH_UNSET,
		"TestSingleKeyStruct": v1.SearchCategory_SEARCH_UNSET,
	}
)
