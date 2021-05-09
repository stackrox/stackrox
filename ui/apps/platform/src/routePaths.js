/**
 * Application route paths constants.
 */

import { resourceTypes, standardEntityTypes, rbacConfigTypes } from 'constants/entityTypes';
import useCaseTypes from 'constants/useCaseTypes';

export const mainPath = '/main';
export const loginPath = '/login';
export const testLoginResultsPath = '/test-login-results';
export const authResponsePrefix = '/auth/response/';

export const dashboardPath = `${mainPath}/dashboard`;
export const networkBasePath = `${mainPath}/network`;
export const networkPath = `${networkBasePath}/:deploymentId?/:externalType?`;
export const violationsBasePath = `${mainPath}/violations`;
export const violationsPath = `${violationsBasePath}/:alertId?`;
export const clustersBasePath = `${mainPath}/clusters`;
export const clustersPathWithParam = `${clustersBasePath}/:clusterId?`;
export const integrationsPath = `${mainPath}/integrations`;
export const policiesBasePath = `${mainPath}/policies`;
export const policiesPath = `${policiesBasePath}/:policyId?/:command?`;
export const riskBasePath = `${mainPath}/risk`;
export const riskPath = `${riskBasePath}/:deploymentId?`;
export const imagesPath = `${mainPath}/images/:imageId?`;
export const secretsPath = `${mainPath}/configmanagement/secrets/:secretId?`;
export const apidocsPath = `${mainPath}/apidocs`;
export const accessControlPath = `${mainPath}/access`;
export const accessControlBasePathV2 = `${mainPath}/access-control`;
export const accessControlPathV2 = `${accessControlBasePathV2}/:entityType?/:entityId?`;
export const userBasePath = `${mainPath}/user`;
export const userRolePath = `${userBasePath}/roles/:roleName`;
export const systemConfigPath = `${mainPath}/systemconfig`;
export const complianceBasePath = `${mainPath}/compliance`;
export const compliancePath = `${mainPath}/:context(compliance)`;
export const configManagementPath = `${mainPath}/configmanagement`;
export const vulnManagementPath = `${mainPath}/vulnerability-management`;
export const dataRetentionPath = `${mainPath}/retention`;
export const systemHealthPath = `${mainPath}/system-health`;
export const productDocsPath = '/docs/product';

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
    [resourceTypes.COMPONENT]: 'components',
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
    [resourceTypes.COMPONENT]: 'component',
    [standardEntityTypes.CONTROL]: 'control',
    [standardEntityTypes.STANDARD]: 'standard',
    [rbacConfigTypes.SERVICE_ACCOUNT]: 'serviceaccount',
    [rbacConfigTypes.SUBJECT]: 'subject',
    [rbacConfigTypes.ROLE]: 'role',
};

export const useCasePaths = {
    [useCaseTypes.VULN_MANAGEMENT]: 'vulnerability-management',
};

export const basePathToLabelMap = {
    [dashboardPath]: 'Dashboard',
    [networkBasePath]: 'Network Graph',
    [violationsBasePath]: 'Violations',
    [complianceBasePath]: 'Compliance',
    [vulnManagementPath]: 'Vulnerability Management',
    [configManagementPath]: 'Configuration Management',
    [riskBasePath]: 'Risk',
    [apidocsPath]: 'API Reference',
    [productDocsPath]: 'Help Center',
    [clustersBasePath]: 'Clusters',
    [policiesBasePath]: 'System Policies',
    [integrationsPath]: 'Integrations',
    [accessControlPath]: 'Access Control',
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
