export const acsAuthNamespaceHeader = 'acs-auth-namespace-scope';

export const metadataRoute = 'metadata';
export const featureFlagsRoute = 'featureFlags';
export const publicConfigRoute = 'publicConfig';
export const getImageCVEListRoute = 'getImageCVEList';

export const deploymentsRoute = 'deployments';
export const getCVEsForDeploymentRoute = 'getCVEsForDeployment';

export const metadataRouteMatcher = { method: 'GET', url: '**/api-service/**/v1/metadata' };
export const featureFlagsRouteMatcher = { method: 'GET', url: '**/api-service/**/v1/featureflags' };
export const publicConfigRouteMatcher = {
    method: 'GET',
    url: '**/api-service/**/v1/config/public',
};

export const routeMatcherMapForBasePlugin = {
    [metadataRoute]: metadataRouteMatcher,
    [featureFlagsRoute]: featureFlagsRouteMatcher,
    [publicConfigRoute]: publicConfigRouteMatcher,
};

export const getImageCVEListRouteMatcher = {
    method: 'POST',
    url: '**/api-service/**/api/graphql?opname=getImageCVEList',
};
export const deploymentListRouteMatcher = {
    method: 'GET',
    url: '**/api-service/**/v1/deployments**',
};
export const getCVEsForDeploymentRouteMatcher = {
    method: 'POST',
    url: '**/api-service/**/api/graphql?opname=getCvesForDeployment',
};
