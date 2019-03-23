/**
 * Application route paths constants.
 */

import { standardTypes, resourceTypes } from 'constants/entityTypes';

export const mainPath = '/main';
export const loginPath = '/login';
export const authResponsePrefix = '/auth/response/';

export const dashboardPath = `${mainPath}/dashboard`;
export const networkPath = `${mainPath}/network`;
export const violationsPath = `${mainPath}/violations/:alertId?`;
export const integrationsPath = `${mainPath}/integrations`;
export const policiesPath = `${mainPath}/policies/:policyId?`;
export const riskPath = `${mainPath}/risk/:deploymentId?`;
export const imagesPath = `${mainPath}/images/:imageId?`;
export const secretsPath = `${mainPath}/secrets/:secretId?`;
export const apidocsPath = `${mainPath}/apidocs`;
export const accessControlPath = `${mainPath}/access`;
export const licensePath = `${mainPath}/license`;

/**
 *Compliance-related route paths
 */
export const resourceTypesToUrl = {
    [resourceTypes.NAMESPACE]: 'namespaces',
    [resourceTypes.CLUSTER]: 'clusters',
    [resourceTypes.NODE]: 'nodes',
    [resourceTypes.DEPLOYMENT]: 'deployments'
};

export const compliancePath = `${mainPath}/compliance`;
const standardsMatcher = `(${Object.values(standardTypes).join('|')})`;

export const nestedCompliancePaths = {
    DASHBOARD: `${compliancePath}/`,
    LIST: `${compliancePath}/:entityType`,
    CONTROL: `${compliancePath}/:standardId${standardsMatcher}/:controlId`,
    CLUSTER: `${compliancePath}/clusters/:entityId`,
    NAMESPACE: `${compliancePath}/namespaces/:entityId`,
    NODE: `${compliancePath}/nodes/:entityId`
};
