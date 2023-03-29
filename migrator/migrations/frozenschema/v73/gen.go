package schema

/***
 * Disable frozen schema generation here. Enabling this only when you need to change and backport a schema change to 3.73.x.
 * Auto generation need to stay disabled after the change.

go:generate pg-schema-migration-helper --type=storage.ActiveComponent --search-category ACTIVE_COMPONENT --references storage.Deployment,storage.ImageComponent
go:generate pg-schema-migration-helper --type=storage.Alert --search-category ALERTS
go:generate pg-schema-migration-helper --type=storage.Alert --search-category ALERTS
go:generate pg-schema-migration-helper --type=storage.AuthProvider --get-all-func
go:generate pg-schema-migration-helper --type=storage.Cluster --search-category CLUSTERS --no-copy-from
go:generate pg-schema-migration-helper --type=storage.ClusterCVE --table=cluster_cves --search-category CLUSTER_VULNERABILITIES --search-scope CLUSTER_VULNERABILITIES,CLUSTER_VULN_EDGE,CLUSTERS
go:generate pg-schema-migration-helper --type=storage.ClusterCVEEdge --table=cluster_cve_edges --search-category CLUSTER_VULN_EDGE --references=storage.Cluster,cluster_cves:storage.ClusterCVE --read-only-store --search-scope CLUSTER_VULNERABILITIES,CLUSTER_VULN_EDGE,CLUSTERS
go:generate pg-schema-migration-helper --type=storage.ClusterHealthStatus --references=storage.Cluster --search-category=CLUSTER_HEALTH
go:generate pg-schema-migration-helper --type=storage.ComplianceDomain --search-category COMPLIANCE_DOMAIN
go:generate pg-schema-migration-helper --type=storage.ComplianceOperatorCheckResult
go:generate pg-schema-migration-helper --type=storage.ComplianceOperatorProfile
go:generate pg-schema-migration-helper --type=storage.ComplianceOperatorRule
go:generate pg-schema-migration-helper --type=storage.ComplianceOperatorScan
go:generate pg-schema-migration-helper --type=storage.ComplianceOperatorScanSettingBinding
go:generate pg-schema-migration-helper --type=storage.ComplianceRunMetadata --search-category COMPLIANCE_METADATA
go:generate pg-schema-migration-helper --type=storage.ComplianceRunResults --search-category COMPLIANCE_RESULTS
go:generate pg-schema-migration-helper --type=storage.ComplianceStrings
go:generate pg-schema-migration-helper --type=storage.ComponentCVEEdge --table=image_component_cve_edges --search-category COMPONENT_VULN_EDGE --references=storage.ImageComponent,image_cves:storage.ImageCVE --read-only-store --search-scope IMAGE_VULNERABILITIES,COMPONENT_VULN_EDGE,IMAGE_COMPONENTS,IMAGE_COMPONENT_EDGE,IMAGE_VULN_EDGE,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS
go:generate pg-schema-migration-helper --type=storage.Config --singleton
go:generate pg-schema-migration-helper --type=storage.Deployment --search-category DEPLOYMENTS --references=storage.Image,namespaces:storage.NamespaceMetadata --search-scope IMAGE_VULNERABILITIES,COMPONENT_VULN_EDGE,IMAGE_COMPONENTS,IMAGE_COMPONENT_EDGE,IMAGE_VULN_EDGE,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS,PROCESS_INDICATORS
go:generate pg-schema-migration-helper --type=storage.ExternalBackup --get-all-func
go:generate pg-schema-migration-helper --type=storage.Group --table=groups --get-all-func
go:generate pg-schema-migration-helper --type=storage.Image --search-category IMAGES --schema-only --search-scope IMAGE_VULNERABILITIES,COMPONENT_VULN_EDGE,IMAGE_COMPONENTS,IMAGE_COMPONENT_EDGE,IMAGE_VULN_EDGE,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS
go:generate pg-schema-migration-helper --type=storage.ImageCVE --table=image_cves --search-category IMAGE_VULNERABILITIES --search-scope IMAGE_VULNERABILITIES,COMPONENT_VULN_EDGE,IMAGE_COMPONENTS,IMAGE_COMPONENT_EDGE,IMAGE_VULN_EDGE,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS
go:generate pg-schema-migration-helper --type=storage.ImageCVEEdge --search-category IMAGE_VULN_EDGE --references=storage.Image,image_cves:storage.ImageCVE --read-only-store --search-scope IMAGE_VULNERABILITIES,COMPONENT_VULN_EDGE,IMAGE_COMPONENTS,IMAGE_COMPONENT_EDGE,IMAGE_VULN_EDGE,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS
go:generate pg-schema-migration-helper --type=storage.ImageComponent --search-category IMAGE_COMPONENTS --search-scope IMAGE_VULNERABILITIES,COMPONENT_VULN_EDGE,IMAGE_COMPONENTS,IMAGE_COMPONENT_EDGE,IMAGE_VULN_EDGE,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS
go:generate pg-schema-migration-helper --type=storage.ImageComponentEdge --search-category IMAGE_COMPONENT_EDGE --references=storage.Image,storage.ImageComponent --read-only-store --search-scope IMAGE_VULNERABILITIES,COMPONENT_VULN_EDGE,IMAGE_COMPONENTS,IMAGE_COMPONENT_EDGE,IMAGE_VULN_EDGE,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS
go:generate pg-schema-migration-helper --type=storage.ImageIntegration --search-category IMAGE_INTEGRATIONS --get-all-func
go:generate pg-schema-migration-helper --type=storage.InitBundleMeta --table=cluster_init_bundles --permission-checker permissionCheckerSingleton()
go:generate pg-schema-migration-helper --type=storage.InstallationInfo --singleton
go:generate pg-schema-migration-helper --type=storage.IntegrationHealth
go:generate pg-schema-migration-helper --type=storage.K8SRole --registered-type=storage.K8sRole --table=k8s_roles --search-category ROLES
go:generate pg-schema-migration-helper --type=storage.K8SRoleBinding --registered-type=storage.K8sRoleBinding --table=role_bindings --search-category ROLEBINDINGS
go:generate pg-schema-migration-helper --type=storage.LogImbue --get-all-func
go:generate pg-schema-migration-helper --type=storage.NamespaceMetadata --table=namespaces --search-category NAMESPACES --references=storage.Cluster --search-scope IMAGE_VULNERABILITIES,COMPONENT_VULN_EDGE,IMAGE_COMPONENTS,IMAGE_COMPONENT_EDGE,IMAGE_VULN_EDGE,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS
go:generate pg-schema-migration-helper --type=storage.NetworkBaseline --search-category NETWORK_BASELINE
go:generate pg-schema-migration-helper --type=storage.NetworkEntity --search-category NETWORK_ENTITY --permission-checker permissionCheckerSingleton()
go:generate pg-schema-migration-helper --type=storage.NetworkGraphConfig
go:generate pg-schema-migration-helper --type=storage.NetworkPolicy --table=networkpolicies --search-category NETWORK_POLICIES
go:generate pg-schema-migration-helper --type=storage.NetworkPolicyApplicationUndoDeploymentRecord --table=networkpoliciesundodeployments
go:generate pg-schema-migration-helper --type=storage.NetworkPolicyApplicationUndoRecord --table=networkpolicyapplicationundorecords
go:generate pg-schema-migration-helper --type=storage.Node --search-category NODES --references=storage.Cluster --schema-only --search-scope NODE_VULNERABILITIES,NODE_COMPONENT_CVE_EDGE,NODE_COMPONENTS,NODE_COMPONENT_EDGE,NODES,CLUSTERS
go:generate pg-schema-migration-helper --type=storage.NodeCVE --table=node_cves --search-category NODE_VULNERABILITIES --search-scope NODE_VULNERABILITIES,NODE_COMPONENT_CVE_EDGE,NODE_COMPONENTS,NODE_COMPONENT_EDGE,NODES,CLUSTERS
go:generate pg-schema-migration-helper --type=storage.NodeComponent --table=node_components --search-category NODE_COMPONENTS --search-scope NODE_VULNERABILITIES,NODE_COMPONENT_CVE_EDGE,NODE_COMPONENTS,NODE_COMPONENT_EDGE,NODES,CLUSTERS
go:generate pg-schema-migration-helper --type=storage.NodeComponentCVEEdge --table=node_components_cves_edges --search-category NODE_COMPONENT_CVE_EDGE --references=node_components:storage.NodeComponent,node_cves:storage.NodeCVE --read-only-store --search-scope NODE_VULNERABILITIES,NODE_COMPONENT_CVE_EDGE,NODE_COMPONENTS,NODE_COMPONENT_EDGE,NODES,CLUSTERS
go:generate pg-schema-migration-helper --type=storage.NodeComponentEdge --search-category NODE_COMPONENT_EDGE --references=storage.Node,node_components:storage.NodeComponent --read-only-store --search-scope NODE_VULNERABILITIES,NODE_COMPONENT_CVE_EDGE,NODE_COMPONENTS,NODE_COMPONENT_EDGE,NODES,CLUSTERS
go:generate pg-schema-migration-helper --type=storage.Notifier --get-all-func
go:generate pg-schema-migration-helper --type=storage.PermissionSet
go:generate pg-schema-migration-helper --type=storage.Pod --search-category PODS --references storage.Deployment
go:generate pg-schema-migration-helper --type=storage.Policy --search-category POLICIES --get-all-func
go:generate pg-schema-migration-helper --type=storage.PolicyCategory --search-category POLICY_CATEGORIES
go:generate pg-schema-migration-helper --type=storage.ProcessBaseline --search-category PROCESS_BASELINES
go:generate pg-schema-migration-helper --type=storage.ProcessBaselineResults --search-category PROCESS_BASELINE_RESULTS
go:generate pg-schema-migration-helper --type=storage.ProcessIndicator --search-category PROCESS_INDICATORS --references storage.Deployment
go:generate pg-schema-migration-helper --type=storage.ReportConfiguration --search-category REPORT_CONFIGURATIONS
go:generate pg-schema-migration-helper --type=storage.ResourceCollection --references=collections:storage.ResourceCollection --table=collections --search-category COLLECTIONS --cycle=EmbeddedCollections
go:generate pg-schema-migration-helper --type=storage.Risk --search-category RISKS
go:generate pg-schema-migration-helper --type=storage.Role
go:generate pg-schema-migration-helper --type=storage.Secret --search-category SECRETS
go:generate pg-schema-migration-helper --type=storage.SensorUpgradeConfig --singleton
go:generate pg-schema-migration-helper --type=storage.ServiceAccount --search-category SERVICE_ACCOUNTS
go:generate pg-schema-migration-helper --type=storage.ServiceIdentity --get-all-func
go:generate pg-schema-migration-helper --type=storage.SignatureIntegration
go:generate pg-schema-migration-helper --type=storage.SimpleAccessScope
go:generate pg-schema-migration-helper --type=storage.TokenMetadata --table=api_tokens
go:generate pg-schema-migration-helper --type=storage.VulnerabilityRequest --search-category VULN_REQUEST --permission-checker permissionCheckerSingleton()
go:generate pg-schema-migration-helper --type=storage.WatchedImage
go:generate sh -c "rm ./convert_*.go"
*/
