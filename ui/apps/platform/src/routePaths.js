/**
 * Application route paths constants.
 */

import { resourceTypes, standardEntityTypes, rbacConfigTypes } from 'constants/entityTypes';

export const mainPath = '/main';
export const loginPath = '/login';
export const testLoginResultsPath = '/test-login-results';
export const authResponsePrefix = '/auth/response/';
export const authorizeRoxctlPath = '/authorize-roxctl';

export const dashboardPath = `${mainPath}/dashboard`;
export const networkBasePath = `${mainPath}/network-graph`;
export const networkPath = `${networkBasePath}/:detailType?/:detailId?`;
export const violationsBasePath = `${mainPath}/violations`;
export const violationsPath = `${violationsBasePath}/:alertId?`;
export const clustersBasePath = `${mainPath}/clusters`;
export const clustersDelegateScanningPath = `${clustersBasePath}/delegate-scanning`;
export const clustersPathWithParam = `${clustersBasePath}/:clusterId?`;
export const clustersListPath = `${mainPath}/clusters-pf`;
export const integrationsPath = `${mainPath}/integrations`;
export const integrationsListPath = `${integrationsPath}/:source/:type`;
export const integrationCreatePath = `${integrationsPath}/:source/:type/create`;
export const integrationDetailsPath = `${integrationsPath}/:source/:type/view/:id`;
export const integrationEditPath = `${integrationsPath}/:source/:type/edit/:id`;
export const policyManagementBasePath = `${mainPath}/policy-management`;
export const policiesBasePath = `${policyManagementBasePath}/policies`;
export const policiesPath = `${policiesBasePath}/:policyId?/:command?`;
export const policyCategoriesPath = `${policyManagementBasePath}/policy-categories`;
export const deprecatedPoliciesBasePath = `${mainPath}/policies`;
export const deprecatedPoliciesPath = `${deprecatedPoliciesBasePath}/:policyId?/:command?`;
export const riskBasePath = `${mainPath}/risk`;
export const riskPath = `${riskBasePath}/:deploymentId?`;
export const secretsPath = `${mainPath}/configmanagement/secrets/:secretId?`;
export const searchPath = `${mainPath}/search`;
export const apidocsPath = `${mainPath}/apidocs`;
export const accessControlBasePath = `${mainPath}/access-control`;
export const accessControlPath = `${accessControlBasePath}/:entitySegment?/:entityId?`;
export const userBasePath = `${mainPath}/user`;
export const userRolePath = `${userBasePath}/roles/:roleName`;
export const systemConfigPath = `${mainPath}/systemconfig`;
export const complianceBasePath = `${mainPath}/compliance`;
export const compliancePath = `${mainPath}/:context(compliance)`;
export const complianceEnhancedBasePath = `${mainPath}/compliance-enhanced`;
export const dataRetentionPath = `${mainPath}/retention`;
export const systemHealthPath = `${mainPath}/system-health`;
export const collectionsBasePath = `${mainPath}/collections`;
export const collectionsPath = `${mainPath}/collections/:collectionId?`;
export const listeningEndpointsBasePath = `${mainPath}/audit/listening-endpoints`;

// Configuration Management

export const configManagementPath = `${mainPath}/configmanagement`;
export const configManagementClustersPath = `${configManagementPath}/clusters`;
export const configManagementControlsPath = `${configManagementPath}/controls`;
export const configManagementDeploymentsPath = `${configManagementPath}/deployments`;
export const configManagementImagesPath = `${configManagementPath}/images`;
export const configManagementNamespacesPath = `${configManagementPath}/namespaces`;
export const configManagementNodesPath = `${configManagementPath}/nodes`;
export const configManagementPoliciesPath = `${configManagementPath}/policies`;
export const configManagementRolesPath = `${configManagementPath}/roles`;
export const configManagementSecretsPath = `${configManagementPath}/secrets`;
export const configManagementServiceAccountsPath = `${configManagementPath}/serviceaccounts`;
export const configManagementSubjectsPath = `${configManagementPath}/subjects`;

// Vuln Management Paths

export const vulnManagementPath = `${mainPath}/vulnerability-management`;
export const vulnManagementPoliciesPath = `${vulnManagementPath}/policies`;
export const vulnManagementCVEsPath = `${vulnManagementPath}/cves`;
export const vulnManagementImageCVEsPath = `${vulnManagementPath}/image-cves`;
export const vulnManagementNodeCVEsPath = `${vulnManagementPath}/node-cves`;
export const vulnManagementPlatformCVEsPath = `${vulnManagementPath}/cluster-cves`;
export const vulnManagementClustersPath = `${vulnManagementPath}/clusters`;
export const vulnManagementNamespacesPath = `${vulnManagementPath}/namespaces`;
export const vulnManagementDeploymentsPath = `${vulnManagementPath}/deployments`;
export const vulnManagementImagesPath = `${vulnManagementPath}/images`;
// TODO: Remove the /components path once we completely split the components into node and image components
export const vulnManagementComponentsPath = `${vulnManagementPath}/components`;
export const vulnManagementNodeComponentsPath = `${vulnManagementPath}/node-components`;
export const vulnManagementImageComponentsPath = `${vulnManagementPath}/image-components`;
export const vulnManagementNodesPath = `${vulnManagementPath}/nodes`;

// The following paths are not part of the infinite nesting Workflow in Vuln Management
export const vulnManagementReportsPath = `${vulnManagementPath}/reports`;
export const vulnManagementReportsPathWithParam = `${vulnManagementPath}/reports/:reportId`;

export const vulnManagementRiskAcceptancePath = `${vulnManagementPath}/risk-acceptance`;
export const vulnManagementPendingApprovalsPath = `${vulnManagementRiskAcceptancePath}/pending-approvals`;
export const vulnManagementApprovedDeferralsPath = `${vulnManagementRiskAcceptancePath}/approved-deferrals`;
export const vulnManagementApprovedFalsePositivesPath = `${vulnManagementRiskAcceptancePath}/approved-false-positives`;

// VM 2.0 "Vulnerabilities" paths
export const vulnerabilitiesBasePath = `${mainPath}/vulnerabilities`;

export const vulnerabilitiesWorkloadCvesPath = `${vulnerabilitiesBasePath}/workload-cves`;
export const vulnerabilitiesWorkloadCveSinglePath = `${vulnerabilitiesBasePath}/workload-cves/cves/:cveId`;
export const vulnerabilitiesWorkloadCveImageSinglePath = `${vulnerabilitiesBasePath}/workload-cves/images/:imageId`;
export const vulnerabilitiesWorkloadCveDeploymentSinglePath = `${vulnerabilitiesBasePath}/workload-cves/deployments/:deploymentId`;

export const vulnerabilityReportsPath = `${vulnerabilitiesBasePath}/reports`;
export const vulnerabilityReportPath = `${vulnerabilitiesBasePath}/reports/:reportId`;

/**
 * New Framwork-related route paths
 */

export const urlEntityListTypes = {
    [resourceTypes.NAMESPACE]: 'namespaces',
    [resourceTypes.CLUSTER]: 'clusters',
    [resourceTypes.NODE]: 'nodes',
    [resourceTypes.DEPLOYMENT]: 'deployments',
    [resourceTypes.IMAGE]: 'images',
    [resourceTypes.SECRET]: 'secrets',
    [resourceTypes.POLICY]: 'policies',
    [resourceTypes.CVE]: 'cves',
    [resourceTypes.IMAGE_CVE]: 'image-cves',
    [resourceTypes.NODE_CVE]: 'node-cves',
    [resourceTypes.CLUSTER_CVE]: 'cluster-cves',
    [resourceTypes.COMPONENT]: 'components',
    [resourceTypes.NODE_COMPONENT]: 'node-components',
    [resourceTypes.IMAGE_COMPONENT]: 'image-components',
    [standardEntityTypes.CONTROL]: 'controls',
    [rbacConfigTypes.SERVICE_ACCOUNT]: 'serviceaccounts',
    [rbacConfigTypes.SUBJECT]: 'subjects',
    [rbacConfigTypes.ROLE]: 'roles',
};

export const urlEntityTypes = {
    [resourceTypes.NAMESPACE]: 'namespace',
    [resourceTypes.CLUSTER]: 'cluster',
    [resourceTypes.NODE]: 'node',
    [resourceTypes.DEPLOYMENT]: 'deployment',
    [resourceTypes.IMAGE]: 'image',
    [resourceTypes.SECRET]: 'secret',
    [resourceTypes.POLICY]: 'policy',
    [resourceTypes.CVE]: 'cve',
    [resourceTypes.IMAGE_CVE]: 'image-cve',
    [resourceTypes.NODE_CVE]: 'node-cve',
    [resourceTypes.CLUSTER_CVE]: 'cluster-cve',
    [resourceTypes.COMPONENT]: 'component',
    [resourceTypes.NODE_COMPONENT]: 'node-component',
    [resourceTypes.IMAGE_COMPONENT]: 'image-component',
    [standardEntityTypes.CONTROL]: 'control',
    [standardEntityTypes.STANDARD]: 'standard',
    [rbacConfigTypes.SERVICE_ACCOUNT]: 'serviceaccount',
    [rbacConfigTypes.SUBJECT]: 'subject',
    [rbacConfigTypes.ROLE]: 'role',
};

const vulnManagementPathToLabelMap = {
    [vulnManagementPath]: 'Dashboard',
    // TODO: add mapping for Deferrals
    [vulnManagementReportsPath]: 'Reporting',
    [vulnManagementRiskAcceptancePath]: 'Risk Acceptance',
};

const vulnerabilitiesPathToLabelMap = {
    [vulnerabilitiesBasePath]: 'Vulnerabilities',
    [vulnerabilitiesWorkloadCvesPath]: 'Workload CVEs',
    [vulnerabilityReportsPath]: 'Vulnerability Reporting',
};

export const basePathToLabelMap = {
    [dashboardPath]: 'Dashboard',
    [networkBasePath]: 'Network Graph (2.0)',
    [violationsBasePath]: 'Violations',
    [complianceBasePath]: 'Compliance',
    [complianceEnhancedBasePath]: 'Compliance (2.0)',
    ...vulnerabilitiesPathToLabelMap,
    ...vulnManagementPathToLabelMap,
    [configManagementPath]: 'Configuration Management',
    [riskBasePath]: 'Risk',
    [apidocsPath]: 'API Reference',
    [clustersBasePath]: 'Clusters',
    [policyManagementBasePath]: 'Policy Management',
    [policiesBasePath]: 'Policy Management',
    [policyCategoriesPath]: 'Policy Categories',
    [collectionsBasePath]: 'Collections',
    [integrationsPath]: 'Integrations',
    [accessControlPath]: 'Access Control',
    [accessControlBasePath]: 'Access Control',
    [systemConfigPath]: 'System Configuration',
    [systemHealthPath]: 'System Health',
    [loginPath]: 'Log In',
    [userBasePath]: 'User Profile',
};

const entityListTypeMatcher = `(${Object.values(urlEntityListTypes).join('|')})`;
const entityTypeMatcher = `(${Object.values(urlEntityTypes).join('|')})`;

export const workflowPaths = {
    DASHBOARD: `${mainPath}/:context`,
    LIST: `${mainPath}/:context/:pageEntityListType${entityListTypeMatcher}/:entityId1?/:entityType2?/:entityId2?`,
    ENTITY: `${mainPath}/:context/:pageEntityType${entityTypeMatcher}/:pageEntityId?/:entityType1?/:entityId1?/:entityType2?/:entityId2?`,
};
