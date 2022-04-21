import * as api from '../constants/apiEndpoints';
import { clustersUrl } from '../constants/ClustersPage';
import { url as dashboardUrl } from '../constants/DashboardPage';
import navigation from '../selectors/navigation';

// Navigation

export function visitClustersFromLeftNav() {
    cy.intercept('POST', api.graphql(api.general.graphqlOps.summaryCounts)).as('getSummaryCounts');
    cy.visit(dashboardUrl);
    cy.wait('@getSummaryCounts');

    cy.intercept('GET', api.clusters.list).as('getClusters');
    cy.get(navigation.navExpandablePlatformConfiguration).click();
    cy.get(
        `${navigation.navExpandablePlatformConfiguration} + ${navigation.nestedNavLinks}:contains("Clusters")`
    ).click();
    cy.wait('@getClusters');
}

export function visitClusters() {
    cy.intercept('GET', api.clusters.list).as('getClusters');
    cy.visit(clustersUrl);
    cy.wait('@getClusters');
}

export function visitClustersWithFixture(fixturePath) {
    cy.intercept('GET', api.clusters.list, {
        fixture: fixturePath,
    }).as('getClusters');
    cy.visit(clustersUrl);
    cy.wait('@getClusters');
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

        cy.visit(`${clustersUrl}/${cluster.id}`);
        cy.wait(['@getClusters', '@getCluster']);
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

        cy.visit(`${clustersUrl}/${cluster.id}`);
        cy.wait(['@getClusters', '@getCluster', '@getMetadata']);
    });
}
