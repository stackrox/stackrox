/**
 * Application route paths constants.
 */

import { standardTypes, resourceTypes } from 'constants/resourceTypes';

export const mainPath = '/main';
export const loginPath = '/login';
export const authResponsePrefix = '/auth/response/';

export const dashboardPath = `${mainPath}/dashboard`;
export const networkPath = `${mainPath}/network`;
export const violationsPath = `${mainPath}/violations/:alertId?`;
export const compliancePath = `${mainPath}/compliance/:clusterId?`;
export const integrationsPath = `${mainPath}/integrations`;
export const policiesPath = `${mainPath}/policies/:policyId?`;
export const riskPath = `${mainPath}/risk/:deploymentId?`;
export const imagesPath = `${mainPath}/images/:imageId?`;
export const secretsPath = `${mainPath}/secrets/:secretId?`;
export const apidocsPath = `${mainPath}/apidocs`;
export const accessControlPath = `${mainPath}/access`;

/**
 *Compliance-related route paths
 */
export const compliance2Path = `${mainPath}/compliance2`;
const standardsMatcher = `(${Object.values(standardTypes).join('|')})`;
const resourcesMatcher = `(${Object.values(resourceTypes).join('|')})`;

export const nestedCompliancePaths = {
    DASHBOARD: `${compliance2Path}/`,
    LIST: `${compliance2Path}/:entityType`,
    CONTROL: `${compliance2Path}/:entityType${standardsMatcher}/:entityId`,
    RESOURCE: `${compliance2Path}/:entityType${resourcesMatcher}/:entityId`
};
