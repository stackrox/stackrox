import { visitFromLeftNavExpandable } from '../../helpers/nav';
import {
    interceptRequests,
    waitForResponses,
    interactAndWaitForResponses,
} from '../../helpers/request';
import { visit } from '../../helpers/visit';

export const sensorUpgradesConfigAlias = 'sensorupgrades/config';
export const clustersAlias = 'clusters';
export const clusterDefaultsAlias = 'cluster-defaults';
export const delegatedRegistryConfigAlias = 'delegatedregistryconfig';
export const delegatedRegistryClustersAlias = `${delegatedRegistryConfigAlias}/clusters`;
export const delegatedRegistryConfigAliasForPUT = 'PUT_delegatedregistryconfig';

const routeMatcherMapForClusterDefaults = {
    [clusterDefaultsAlias]: {
        method: 'GET',
        url: '/v1/cluster-defaults',
    },
};

const routeMatcherMapForClusters = {
    [sensorUpgradesConfigAlias]: {
        method: 'GET',
        url: '/v1/sensorupgrades/config',
    },
    [clustersAlias]: {
        method: 'GET',
        url: 'v1/clusters',
    },
    ...routeMatcherMapForClusterDefaults,
};
const routeMatcherMapForDelegateScanning = {
    [delegatedRegistryConfigAlias]: {
        method: 'GET',
        url: '/v1/delegatedregistryconfig',
    },
    [delegatedRegistryClustersAlias]: {
        method: 'GET',
        url: '/v1/delegatedregistryconfig/clusters',
    },
};

const basePath = '/main/clusters';
export const delegatedScanningPath = `${basePath}/delegated-image-scanning`;

const title = 'Clusters';

// assert

export function assertClusterNameInSidePanel(clusterName) {
    cy.get(`[data-testid="clusters-side-panel-header"]:contains("${clusterName}")`);
}

// visit

/**
 * Visit clusters by interaction from another container.
 * For example, click View All button from System Health.
 *
 * @param {function} interactionCallback
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function interactAndVisitClusters(interactionCallback, staticResponseMap) {
    interceptRequests(routeMatcherMapForClusters, staticResponseMap);

    interactionCallback();

    cy.location('pathname').should('eq', basePath);
    cy.get(`h1:contains("${title}")`);

    waitForResponses(routeMatcherMapForClusters);
}

export function visitClustersFromLeftNav() {
    visitFromLeftNavExpandable('Platform Configuration', title, routeMatcherMapForClusters);

    cy.location('pathname').should('eq', basePath);
    cy.get(`h1:contains("${title}")`);
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitClusters(staticResponseMap) {
    visit(basePath, routeMatcherMapForClusters, staticResponseMap);

    cy.get(`h1:contains("${title}")`);
}

export function visitClustersWithFixture(fixturePath) {
    visitClusters({
        [clustersAlias]: { fixture: fixturePath },
    });
}

export const clusterAlias = 'clusters/id';

/**
 * @param {string} clusterId
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitClusterById(clusterId, staticResponseMap) {
    const routeMatcherMapForClusterById = {
        ...routeMatcherMapForClusterDefaults,
        [clusterAlias]: {
            method: 'GET',
            url: `/v1/clusters/${clusterId}`,
        },
    };
    visit(`${basePath}/${clusterId}`, routeMatcherMapForClusterById, staticResponseMap);

    cy.get(`h1:contains("${title}")`);
}

export function visitClustersWithFixtureMetadataDatetime(fixturePath, metadata, datetimeISOString) {
    cy.intercept('GET', 'v1/metadata', {
        body: metadata,
    }).as('metadata');

    // For comparison to `lastContact` and `sensorCertExpiry` in clusters fixture.
    const currentDatetime = new Date(datetimeISOString);
    cy.clock(currentDatetime.getTime(), ['Date', 'setInterval']);

    visitClustersWithFixture(fixturePath);

    cy.wait('@metadata');
}

export function visitClusterByNameWithFixture(clusterName, fixturePath) {
    cy.fixture(fixturePath).then(({ clusters, clusterIdToRetentionInfo }) => {
        const cluster = clusters.find(({ name }) => name === clusterName);
        const clusterRetentionInfo = clusterIdToRetentionInfo[cluster.id] ?? null;

        visitClusterById(cluster.id, {
            [clusterAlias]: { body: { cluster, clusterRetentionInfo } },
        });

        assertClusterNameInSidePanel(clusterName);
    });
}

export function visitClusterByNameWithFixtureMetadataDatetime(
    clusterName,
    fixturePath,
    metadata,
    datetimeISOString
) {
    cy.fixture(fixturePath).then(({ clusters, clusterIdToRetentionInfo }) => {
        cy.intercept('GET', 'v1/metadata', {
            body: metadata,
        }).as('metadata');

        const cluster = clusters.find(({ name }) => name === clusterName);
        const clusterRetentionInfo = clusterIdToRetentionInfo[cluster.id] ?? null;

        // For comparison to `lastContact` and `sensorCertExpiry` in clusters fixture.
        const currentDatetime = new Date(datetimeISOString);
        cy.clock(currentDatetime.getTime(), ['Date', 'setInterval']);

        visitClusterById(cluster.id, {
            [clusterAlias]: { body: { cluster, clusterRetentionInfo } },
        });

        cy.wait(['@metadata']);
        assertClusterNameInSidePanel(clusterName);
    });
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitDelegateScanning(staticResponseMap) {
    visit(delegatedScanningPath, routeMatcherMapForDelegateScanning, staticResponseMap);
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function saveDelegatedRegistryConfig(staticResponseMap) {
    const routeMatcherMap = {
        [delegatedRegistryConfigAliasForPUT]: {
            method: 'PUT',
            url: '/v1/delegatedregistryconfig',
        },
    };

    return interactAndWaitForResponses(
        () => {
            cy.get('button:contains("Save")').click();
        },
        routeMatcherMap,
        staticResponseMap
    );
}
