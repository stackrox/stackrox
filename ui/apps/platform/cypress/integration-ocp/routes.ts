import { interceptAndWatchRequests, interceptRequests } from '../helpers/request';
import { hasFeatureFlag } from '../helpers/features';
import { toImageV2Response } from '../integration/vulnerabilities/workloadCves/WorkloadCves.helpers';

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
    const isFlattenImageData = hasFeatureFlag('ROX_FLATTEN_IMAGE_DATA');

    interceptOcpGraphQL({
        getImageCVEList: { fixture: 'vulnerabilities/workloadCves/getImageCVEList.json' },
        getImageCveMetadata: { fixture: 'vulnerabilities/workloadCves/getImageCveMetadata.json' },
        getImageCveSummaryData: {
            fixture: 'vulnerabilities/workloadCves/getImageCveSummaryData.json',
        },
        getImagesForCVE: { fixture: 'vulnerabilities/workloadCves/getImagesForCVE.json' },
        getImageList: { fixture: 'vulnerabilities/workloadCves/getImageList.json' },
    });

    // When ROX_FLATTEN_IMAGE_DATA is enabled, queries return ImageV2 types instead of
    // Image types. Fixtures use the v1 shape, so we transform them for v2 compatibility.
    // getImageDetails uses an alias (image: imageV2) so the root key stays `image`,
    // but __typename must be ImageV2 for the fragment to match.
    // getCVEsForImage and getImageResources use `imageV2` as the root key directly.
    if (isFlattenImageData) {
        cy.fixture('vulnerabilities/workloadCves/imageWithMultipleCves.json').then((v1Response) => {
            interceptOcpGraphQL({
                getImageDetails: {
                    body: {
                        data: {
                            image: {
                                ...v1Response.data.image,
                                __typename: 'ImageV2',
                            },
                        },
                    },
                },
            });
        });

        const imageV2Fixtures = [
            {
                opname: 'getCVEsForImage',
                fixture: 'vulnerabilities/workloadCves/multipleCvesForImage.json',
            },
            {
                opname: 'getImageResources',
                fixture: 'vulnerabilities/workloadCves/getImageResources.json',
            },
        ];

        imageV2Fixtures.forEach(({ opname, fixture }) => {
            cy.fixture(fixture).then((v1Response) => {
                interceptOcpGraphQL({ [opname]: { body: toImageV2Response(v1Response) } });
            });
        });
    } else {
        interceptOcpGraphQL({
            getImageDetails: { fixture: 'vulnerabilities/workloadCves/imageWithMultipleCves.json' },
            getCVEsForImage: { fixture: 'vulnerabilities/workloadCves/multipleCvesForImage.json' },
            getImageResources: { fixture: 'vulnerabilities/workloadCves/getImageResources.json' },
        });
    }
}
