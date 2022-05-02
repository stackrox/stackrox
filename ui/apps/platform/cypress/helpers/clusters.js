import * as api from '../constants/apiEndpoints';
import { clustersUrl, selectors } from '../constants/ClustersPage';

import { visitFromLeftNavExpandable } from './nav';
import { visit } from './visit';

// Navigation

export function visitClustersFromLeftNav() {
    cy.intercept('GET', api.clusters.list).as('getClusters');
    visitFromLeftNavExpandable('Platform Configuration', 'Clusters');
    cy.wait('@getClusters');
    cy.get(selectors.clustersListHeading).contains('Clusters');
}

export function visitClusters() {
    cy.intercept('GET', api.clusters.list).as('getClusters');
    visit(clustersUrl);
    cy.wait('@getClusters');
    cy.get(selectors.clustersListHeading).contains('Clusters');
}

export function visitClustersWithFixture(fixturePath) {
    cy.intercept('GET', api.clusters.list, {
        fixture: fixturePath,
    }).as('getClusters');
    visit(clustersUrl);
    cy.wait('@getClusters');
    cy.get(selectors.clustersListHeading).contains('Clusters');
}

export function visitClustersWithFixtureMetadataDatetime(fixturePath, metadata, datetimeISOString) {
    cy.intercept('GET', api.metadata, {
        body: metadata,
    }).as('getMetadata');

    // For comparison to `lastContact` and `sensorCertExpiry` in clusters fixture.
    const currentDatetime = new Date(datetimeISOString);
    cy.clock(currentDatetime.getTime(), ['Date', 'setInterval']);

    visitClustersWithFixture(fixturePath);

    cy.wait('@getMetadata');
}

export function visitClusterByNameWithFixture(clusterName, fixturePath) {
    cy.fixture(fixturePath).then(({ clusters }) => {
        cy.intercept('GET', api.clusters.list, {
            body: { clusters },
        }).as('getClusters');

        const cluster = clusters.find(({ name }) => name === clusterName);
        cy.intercept('GET', api.clusters.single, {
            body: { cluster },
        }).as('getCluster');

        visit(`${clustersUrl}/${cluster.id}`);
        cy.wait(['@getClusters', '@getCluster']);
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
        cy.intercept('GET', api.clusters.list, {
            body: { clusters },
        }).as('getClusters');
        cy.intercept('GET', api.metadata, {
            body: metadata,
        }).as('getMetadata');

        const cluster = clusters.find(({ name }) => name === clusterName);
        cy.intercept('GET', api.clusters.single, {
            body: { cluster },
        }).as('getCluster');

        // For comparison to `lastContact` and `sensorCertExpiry` in clusters fixture.
        const currentDatetime = new Date(datetimeISOString);
        cy.clock(currentDatetime.getTime(), ['Date', 'setInterval']);

        visit(`${clustersUrl}/${cluster.id}`);
        cy.wait(['@getClusters', '@getCluster', '@getMetadata']);
        cy.get(selectors.clusterSidePanelHeading).contains(clusterName);
    });
}
