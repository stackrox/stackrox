/**
 * Application route paths constants.
 */

import { resourceTypes, standardEntityTypes, rbacConfigTypes } from 'constants/entityTypes';
import useCaseTypes from 'constants/useCaseTypes';

export const mainPath = '/main';
export const loginPath = '/login';
export const licenseStartUpPath = `/license`;
export const authResponsePrefix = '/auth/response/';

export const dashboardPath = `${mainPath}/dashboard`;
export const networkPath = `${mainPath}/network/:deploymentId?`;
export const violationsPath = `${mainPath}/violations/:alertId?`;
export const clustersPath = `${mainPath}/clusters/:clusterId?`;
export const integrationsPath = `${mainPath}/integrations`;
export const policiesListPath = `${mainPath}/policies`;
export const policiesPath = `${policiesListPath}/:policyId?/:command?`;
export const riskPath = `${mainPath}/risk/:deploymentId?`;
export const imagesPath = `${mainPath}/images/:imageId?`;
export const secretsPath = `${mainPath}/configmanagement/secrets/:secretId?`;
export const apidocsPath = `${mainPath}/apidocs`;
export const accessControlPath = `${mainPath}/access`;
export const licensePath = `${mainPath}/license`;
export const systemConfigPath = `${mainPath}/systemconfig`;
export const compliancePath = `${mainPath}/:context(compliance)`;
export const configManagementPath = `${mainPath}/configmanagement`;
export const vulnManagementPath = `${mainPath}/vulnerability-management`;
export const dataRetentionPath = `${mainPath}/retention`;

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
    [rbacConfigTypes.ROLE]: 'roles'
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
    [rbacConfigTypes.ROLE]: 'role'
};

export const useCasePaths = {
    [useCaseTypes.VULN_MANAGEMENT]: 'vulnerability-management'
};

const entityListTypeMatcher = `(${Object.values(urlEntityListTypes).join('|')})`;
const entityTypeMatcher = `(${Object.values(urlEntityTypes).join('|')})`;

export const nestedPaths = {
    DASHBOARD: `${mainPath}/:context`,
    LIST: `${mainPath}/:context/:pageEntityListType${entityListTypeMatcher}/:entityId1?/:entityType2?/:entityId2?`,
    ENTITY: `${mainPath}/:context/:pageEntityType${entityTypeMatcher}/:pageEntityId?/:entityType1?/:entityId1?/:entityType2?/:entityId2?`
};
