import * as api from '../constants/apiEndpoints';
import { clustersUrl, selectors } from '../constants/ClustersPage';

import { visitFromLeftNavExpandable } from './nav';
import { visit } from './visit';

const routeMatcherMap = {
    'sensorupgrades/config': {
        method: 'GET',
        url: api.clusters.sensorUpgradesConfig,
    },
    clusters: {
        method: 'GET',
        url: api.clusters.list,
    },
    'cluster-defaults': {
        method: 'GET',
        url: api.clusters.clusterDefaults,
    },
};

// Navigation

export function visitClustersFromLeftNav() {
    visitFromLeftNavExpandable('Platform Configuration', 'Clusters', { routeMatcherMap });

    cy.get(selectors.clustersListHeading).contains('Clusters');
}

export function visitClusters(staticResponseMap) {
    visit(clustersUrl, { routeMatcherMap }, staticResponseMap);

    cy.get(selectors.clustersListHeading).contains('Clusters');
}

export function visitClusterById(clusterId, staticResponseMap) {
    const routeMatcherMapClusterById = {
        ...routeMatcherMap,
        cluster: {
            method: 'GET',
            url: api.clusters.single,
        },
    };
    visit(
        `${clustersUrl}/${clusterId}`,
        { routeMatcherMap: routeMatcherMapClusterById },
        staticResponseMap
    );

    cy.get(selectors.clustersListHeading).contains('Clusters');
}

export function visitClustersWithFixtureMetadataDatetime(fixturePath, metadata, datetimeISOString) {
    cy.intercept('GET', api.metadata, {
        body: metadata,
    }).as('metadata');

    // For comparison to `lastContact` and `sensorCertExpiry` in clusters fixture.
    const currentDatetime = new Date(datetimeISOString);
    cy.clock(currentDatetime.getTime(), ['Date', 'setInterval']);

    visitClusters({
        clusters: { fixture: fixturePath },
    });

    cy.wait('@metadata');
}

export function visitClusterByNameWithFixture(clusterName, fixturePath) {
    cy.fixture(fixturePath).then(({ clusters }) => {
        const cluster = clusters.find(({ name }) => name === clusterName);

        visitClusterById(cluster.id, {
            clusters: { body: { clusters } },
            cluster: { body: { cluster } },
        });

        cy.get(selectors.clusterSidePanelHeading).contains(clusterName);
    });
}

export function visitClusterByNameWithFixtureMetadataDatetime(
    clusterName,
    fixturePath,
    metadata,
    datetimeISOString
) {
    cy.fixture(fixturePath).then(({ clusters }) => {
        cy.intercept('GET', api.metadata, {
            body: metadata,
        }).as('metadata');

        const cluster = clusters.find(({ name }) => name === clusterName);

        // For comparison to `lastContact` and `sensorCertExpiry` in clusters fixture.
        const currentDatetime = new Date(datetimeISOString);
        cy.clock(currentDatetime.getTime(), ['Date', 'setInterval']);

        visitClusterById(cluster.id, {
            clusters: { body: { clusters } },
            cluster: { body: { cluster } },
        });

        cy.wait(['@metadata']);
        cy.get(selectors.clusterSidePanelHeading).contains(clusterName);
    });
}
