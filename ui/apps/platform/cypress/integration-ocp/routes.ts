import { interceptAndWatchRequests, interceptRequests } from '../helpers/request';

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

export function getOcpRouteMatcherMapForGraphQL<T extends string>(opnames: T[]) {
    return Object.fromEntries(
        opnames.map((opname) => [
            opname,
            { method: 'POST' as const, url: `**/api-service/**/api/graphql?opname=${opname}` },
        ])
    ) as Record<T, { method: 'POST'; url: string }>;
}

type FixtureMap = Record<string, { fixture: string } | { body: unknown }>;

export function interceptOcpGraphQL(fixtureMap: FixtureMap) {
    interceptRequests(getOcpRouteMatcherMapForGraphQL(Object.keys(fixtureMap)), fixtureMap);
}

export function watchOcpGraphQL(fixtureMap: FixtureMap) {
    return interceptAndWatchRequests(
        getOcpRouteMatcherMapForGraphQL(Object.keys(fixtureMap)),
        fixtureMap
    );
}

export function interceptWorkloadCveFixtures() {
    interceptOcpGraphQL({
        getCVEsForImage: { fixture: 'vulnerabilities/workloadCves/multipleCvesForImage.json' },
        getImageCVEList: { fixture: 'vulnerabilities/workloadCves/getImageCVEList.json' },
        getImageCveMetadata: { fixture: 'vulnerabilities/workloadCves/getImageCveMetadata.json' },
        getImageCveSummaryData: {
            fixture: 'vulnerabilities/workloadCves/getImageCveSummaryData.json',
        },
        getImageDetails: { fixture: 'vulnerabilities/workloadCves/imageWithMultipleCves.json' },
        getImageResources: { fixture: 'vulnerabilities/workloadCves/getImageResources.json' },
        getImagesForCVE: { fixture: 'vulnerabilities/workloadCves/getImagesForCVE.json' },
        getImageList: { fixture: 'vulnerabilities/workloadCves/getImageList.json' },
    });
}
