import * as api from '../constants/apiEndpoints';
import { clustersUrl, selectors } from '../constants/ClustersPage';

import { visitFromLeftNavExpandable } from './nav';
import { interceptAndWaitForResponses } from './request';
import { visit } from './visit';

export const sensorUpgradesConfigAlias = 'sensorupgrades/config';
export const clustersAlias = 'clusters';
export const clusterDefaultsAlias = 'cluster-defaults';

const routeMatcherMap = {
    [sensorUpgradesConfigAlias]: {
        method: 'GET',
        url: api.clusters.sensorUpgradesConfig,
    },
    [clustersAlias]: {
        method: 'GET',
        url: api.clusters.list,
    },
    [clusterDefaultsAlias]: {
        method: 'GET',
        url: api.clusters.clusterDefaults,
    },
};

const title = 'Clusters';

// Navigation

/**
 * Reach clusters by interaction from another container.
 * For example, click View All button from System Health.
 *
 * @param {function} interactionCallback
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function reachClusters(interactionCallback, staticResponseMap) {
    interactionCallback();

    cy.location('pathname').should('eq', clustersUrl);
    cy.get(`h1:contains("${title}")`);

    interceptAndWaitForResponses(routeMatcherMap, staticResponseMap);
}

export function visitClustersFromLeftNav() {
    visitFromLeftNavExpandable('Platform Configuration', title);

    cy.location('pathname').should('eq', clustersUrl);
    cy.get(`h1:contains("${title}")`);

    interceptAndWaitForResponses(routeMatcherMap);
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitClusters(staticResponseMap) {
    visit(clustersUrl);

    cy.get(`h1:contains("${title}")`);

    interceptAndWaitForResponses(routeMatcherMap, routeMatcherMap, staticResponseMap);
}

export function visitClustersWithFixture(fixturePath) {
    const staticResponseMap = {
        [clustersAlias]: {
            fixture: fixturePath,
        },
    };

    visitClusters(staticResponseMap);
}

export const clusterAlias = 'clusters/id';

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitClusterById(clusterId, staticResponseMap) {
    const routeMatcherMapClusterById = {
        [clusterAlias]: {
            method: 'GET',
            url: `${api.clusters.list}/${clusterId}`,
        },
        [clusterDefaultsAlias]: {
            method: 'GET',
            url: api.clusters.clusterDefaults,
        },
    };

    visit(`${clustersUrl}/${clusterId}`);

    cy.get(`h1:contains("${title}")`); // update assertion when cluster page replaces side panel.

    interceptAndWaitForResponses(routeMatcherMapClusterById, staticResponseMap);
}

export function visitClustersWithFixtureMetadataDatetime(fixturePath, metadata, datetimeISOString) {
    cy.intercept('GET', api.metadata, {
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

        const staticResponseMap = {
            [clusterAlias]: {
                body: { cluster, clusterRetentionInfo },
            },
        };

        visitClusterById(cluster.id, staticResponseMap);

        cy.get(selectors.clusterSidePanelHeading).contains(clusterName);
    });
}

export function visitClusterByNameWithFixtureMetadataDatetime(
    clusterName,
    fixturePath,
    metadata,
    datetimeISOString
) {
    cy.fixture(fixturePath).then(({ clusters, clusterIdToRetentionInfo }) => {
        cy.intercept('GET', api.metadata, {
            body: metadata,
        }).as('metadata');

        const cluster = clusters.find(({ name }) => name === clusterName);
        const clusterRetentionInfo = clusterIdToRetentionInfo[cluster.id] ?? null;

        // For comparison to `lastContact` and `sensorCertExpiry` in clusters fixture.
        const currentDatetime = new Date(datetimeISOString);
        cy.clock(currentDatetime.getTime(), ['Date', 'setInterval']);

        const staticResponseMap = {
            [clusterAlias]: {
                body: { cluster, clusterRetentionInfo },
            },
        };

        visitClusterById(cluster.id, staticResponseMap);

        cy.wait(['@metadata']);
        cy.get(selectors.clusterSidePanelHeading).contains(clusterName);
    });
}

export function visitDashboardWithNoClusters() {
    cy.intercept('POST', api.graphql('summary_counts'), {
        body: {
            data: {
                clusterCount: 0,
                nodeCount: 3,
                violationCount: 20,
                deploymentCount: 35,
                imageCount: 31,
                secretCount: 15,
            },
        },
    }).as('summary_counts');
    cy.intercept('GET', api.clusters.list, {
        clusters: [],
    }).as('clusters');

    // visitMainDashboard(); // with a count of 0 clusters, app should redirect to the clusters pages
    cy.visit('/main/dashboard'); // with a count of 0 clusters, app should redirect to the clusters pages

    cy.wait(['@summary_counts', '@clusters']);
}
