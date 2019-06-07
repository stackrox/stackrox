/**
 * Application route paths constants.
 */

import {
    standardTypes,
    resourceTypes,
    standardEntityTypes,
    rbacConfigTypes
} from 'constants/entityTypes';

export const mainPath = '/main';
export const loginPath = '/login';
export const licenseStartUpPath = `/license`;
export const authResponsePrefix = '/auth/response/';

export const dashboardPath = `${mainPath}/dashboard`;
export const networkPath = `${mainPath}/network/:deploymentId?`;
export const violationsPath = `${mainPath}/violations/:alertId?`;
export const integrationsPath = `${mainPath}/integrations`;
export const policiesPath = `${mainPath}/policies/:policyId?`;
export const riskPath = `${mainPath}/risk/:deploymentId?`;
export const imagesPath = `${mainPath}/images/:imageId?`;
export const secretsPath = `${mainPath}/secrets/:secretId?`;
export const apidocsPath = `${mainPath}/apidocs`;
export const accessControlPath = `${mainPath}/access`;
export const licensePath = `${mainPath}/license`;
export const systemConfigPath = `${mainPath}/systemconfig`;

/**
 *Compliance-related route paths
 */

export const resourceTypesToUrl = {
    [resourceTypes.NAMESPACE]: 'namespaces',
    [resourceTypes.CLUSTER]: 'clusters',
    [resourceTypes.NODE]: 'nodes',
    [resourceTypes.DEPLOYMENT]: 'deployments',
    [standardEntityTypes.CONTROL]: 'controls'
};

export const compliancePath = `${mainPath}/compliance`;
const standardsMatcher = `(${Object.values(standardTypes).join('|')})`;
const resourceMatcher = `(${Object.values(resourceTypesToUrl).join('|')})`;

export const nestedCompliancePaths = {
    DASHBOARD: `${compliancePath}/`,
    LIST: `${compliancePath}/:entityType`,
    CONTROL: `${compliancePath}/:standardId${standardsMatcher}/:controlId/:listEntityType${resourceMatcher}?`,
    CLUSTER: `${compliancePath}/clusters/:entityId/:listEntityType${resourceMatcher}?`,
    NAMESPACE: `${compliancePath}/namespaces/:entityId/:listEntityType${resourceMatcher}?`,
    DEPLOYMENT: `${compliancePath}/deployments/:entityId/:listEntityType${resourceMatcher}?`,
    NODE: `${compliancePath}/nodes/:entityId/:listEntityType${resourceMatcher}?`
};

export const urlEntityListTypes = {
    [resourceTypes.NAMESPACE]: 'namespaces',
    [resourceTypes.CLUSTER]: 'clusters',
    [resourceTypes.NODE]: 'nodes',
    [resourceTypes.DEPLOYMENT]: 'deployments',
    [resourceTypes.IMAGE]: 'images',
    [resourceTypes.SECRET]: 'secrets',
    [resourceTypes.POLICY]: 'policies',
    [standardEntityTypes.CONTROL]: 'controls',
    [rbacConfigTypes.SERVICE_ACCOUNT]: 'serviceaccounts',
    [rbacConfigTypes.SUBJECT]: 'subjects',
    [rbacConfigTypes.ROLE]: 'roles'
};

/**
 * New Framwork-related route paths
 */

export const urlEntityTypes = {
    [resourceTypes.NAMESPACE]: 'namespace',
    [resourceTypes.CLUSTER]: 'cluster',
    [resourceTypes.NODE]: 'node',
    [resourceTypes.DEPLOYMENT]: 'deployment',
    [resourceTypes.IMAGE]: 'image',
    [resourceTypes.SECRET]: 'secret',
    [resourceTypes.POLICY]: 'policy',
    [standardEntityTypes.CONTROL]: 'control',
    [standardEntityTypes.STANDARD]: 'standard',
    [rbacConfigTypes.SERVICE_ACCOUNT]: 'serviceaccount',
    [rbacConfigTypes.SUBJECT]: 'subject',
    [rbacConfigTypes.ROLE]: 'role'
};

export const configManagementPath = `${mainPath}/configmanagement`;

const entityListTypeMatcher = `(${Object.values(urlEntityListTypes).join('|')})`;
const entityTypeMatcher = `(${Object.values(urlEntityTypes).join('|')})`;

export const nestedPaths = {
    DASHBOARD: `${mainPath}/:context(configmanagement)`,
    LIST: `/:pageEntityListType${entityListTypeMatcher}/:entityId1?/:entityType2?/:entityId2?`,
    ENTITY: `/:pageEntityType${entityTypeMatcher}/:pageEntityId?/:entityListType1${entityListTypeMatcher}?/:entityId1?/:entityType2?/:entityId2?`
};
