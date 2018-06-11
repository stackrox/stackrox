/**
 * Application route paths constants.
 */

export const mainPath = '/main';
export const loginPath = '/login';
export const oidcResponsePath = '/auth/response/oidc';

export const dashboardPath = `${mainPath}/dashboard`;
export const violationsPath = `${mainPath}/violations/:alertId?`;
export const compliancePath = `${mainPath}/compliance/:clusterId?`;
export const integrationsPath = `${mainPath}/integrations`;
export const policiesPath = `${mainPath}/policies/:policyId?`;
export const riskPath = `${mainPath}/risk/:deploymentId?`;
export const imagesPath = `${mainPath}/images/:imageSha?`;
