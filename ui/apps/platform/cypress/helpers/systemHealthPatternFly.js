import * as api from '../constants/apiEndpoints';
import { systemHealthUrl } from '../constants/SystemHealthPatternFly';

import { visitFromLeftNavExpandable } from './nav';
import { visit } from './visit';

// clock

// Call before visit function.
export function setClock(currentDatetime) {
    cy.clock(currentDatetime.getTime(), ['Date']);
}

// visit

export function visitSystemHealthFromLeftNav() {
    cy.intercept('GET', api.clusters.list).as('getClusters');
    cy.intercept('GET', api.integrations.externalBackups).as('getBackupIntegrations');
    cy.intercept('GET', api.integrations.imageIntegrations).as('getImageIntegrations');
    cy.intercept('GET', api.integrations.notifiers).as('getNotifierIntegrations');
    cy.intercept('GET', api.integrationHealth.externalBackups).as('getBackupIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.imageIntegrations).as('getImageIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.notifiers).as('getNotifierIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.vulnDefinitions).as('getVulnDefinitionsHealth');

    visitFromLeftNavExpandable('Platform Configuration', 'System Health');

    cy.wait([
        '@getClusters',
        '@getBackupIntegrations',
        '@getImageIntegrations',
        '@getNotifierIntegrations',
        '@getBackupIntegrationsHealth',
        '@getImageIntegrationsHealth',
        '@getNotifierIntegrationsHealth',
        '@getVulnDefinitionsHealth',
    ]);
    cy.get('h1:contains("System Health")');
}

export function visitSystemHealth() {
    // cy.intercept('GET', api.clusters.list).as('getClusters');
    cy.intercept('GET', api.integrations.externalBackups).as('getBackupIntegrations');
    cy.intercept('GET', api.integrations.imageIntegrations).as('getImageIntegrations');
    cy.intercept('GET', api.integrations.notifiers).as('getNotifierIntegrations');
    cy.intercept('GET', api.integrationHealth.externalBackups).as('getBackupIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.imageIntegrations).as('getImageIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.notifiers).as('getNotifierIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.vulnDefinitions).as('getVulnDefinitionsHealth');

    visit(systemHealthUrl);

    cy.wait([
        // '@getClusters',
        '@getBackupIntegrations',
        '@getImageIntegrations',
        '@getNotifierIntegrations',
        '@getBackupIntegrationsHealth',
        '@getImageIntegrationsHealth',
        '@getNotifierIntegrationsHealth',
        '@getVulnDefinitionsHealth',
    ]);
    cy.get('h1:contains("System Health")');
}

// visit clusters

export function visitSystemHealthWithClustersFixture(fixturePath) {
    cy.intercept('GET', api.clusters.list, {
        fixture: fixturePath,
    }).as('getClusters');
    cy.intercept('GET', api.integrations.externalBackups).as('getBackupIntegrations');
    cy.intercept('GET', api.integrations.imageIntegrations).as('getImageIntegrations');
    cy.intercept('GET', api.integrations.notifiers).as('getNotifierIntegrations');
    cy.intercept('GET', api.integrationHealth.externalBackups).as('getBackupIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.imageIntegrations).as('getImageIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.notifiers).as('getNotifierIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.vulnDefinitions).as('getVulnDefinitionsHealth');

    visit(systemHealthUrl);

    cy.wait([
        '@getClusters',
        '@getBackupIntegrations',
        '@getImageIntegrations',
        '@getNotifierIntegrations',
        '@getBackupIntegrationsHealth',
        '@getImageIntegrationsHealth',
        '@getNotifierIntegrationsHealth',
        '@getVulnDefinitionsHealth',
    ]);
    cy.get('h1:contains("System Health")');
}

export function visitSystemHealthWithClustersFixtureFilteredByNames(fixturePath, clusterNames) {
    cy.fixture(fixturePath).then(({ clusters }) => {
        cy.intercept('GET', api.clusters.list, {
            body: { clusters: clusters.filter(({ name }) => clusterNames.includes(name)) },
        }).as('getClusters');
        cy.intercept('GET', api.integrations.externalBackups).as('getBackupIntegrations');
        cy.intercept('GET', api.integrations.imageIntegrations).as('getImageIntegrations');
        cy.intercept('GET', api.integrations.notifiers).as('getNotifierIntegrations');
        cy.intercept('GET', api.integrationHealth.externalBackups).as(
            'getBackupIntegrationsHealth'
        );
        cy.intercept('GET', api.integrationHealth.imageIntegrations).as(
            'getImageIntegrationsHealth'
        );
        cy.intercept('GET', api.integrationHealth.notifiers).as('getNotifierIntegrationsHealth');
        cy.intercept('GET', api.integrationHealth.vulnDefinitions).as('getVulnDefinitionsHealth');

        visit(systemHealthUrl);

        cy.wait([
            '@getClusters',
            '@getBackupIntegrations',
            '@getImageIntegrations',
            '@getNotifierIntegrations',
            '@getBackupIntegrationsHealth',
            '@getImageIntegrationsHealth',
            '@getNotifierIntegrationsHealth',
            '@getVulnDefinitionsHealth',
        ]);
        cy.get('h1:contains("System Health")');
    });
}

// visit integrations

export function visitSystemHealthWithBackupIntegrations(externalBackups, integrationHealth) {
    cy.intercept('GET', api.clusters.list).as('getClusters');
    cy.intercept('GET', api.integrations.externalBackups, {
        body: { externalBackups },
    }).as('getBackupIntegrations');
    cy.intercept('GET', api.integrations.imageIntegrations).as('getImageIntegrations');
    cy.intercept('GET', api.integrations.notifiers).as('getNotifierIntegrations');
    cy.intercept('GET', api.integrationHealth.externalBackups, {
        body: { integrationHealth },
    }).as('getBackupIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.imageIntegrations).as('getImageIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.notifiers).as('getNotifierIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.vulnDefinitions).as('getVulnDefinitionsHealth');

    visit(systemHealthUrl);

    cy.wait([
        '@getClusters',
        '@getBackupIntegrations',
        '@getImageIntegrations',
        '@getNotifierIntegrations',
        '@getBackupIntegrationsHealth',
        '@getImageIntegrationsHealth',
        '@getNotifierIntegrationsHealth',
        '@getVulnDefinitionsHealth',
    ]);
    cy.get('h1:contains("System Health")');
}

export function visitSystemHealthWithImageIntegrations(integrations, integrationHealth) {
    cy.intercept('GET', api.clusters.list).as('getClusters');
    cy.intercept('GET', api.integrations.externalBackups).as('getBackupIntegrations');
    cy.intercept('GET', api.integrations.imageIntegrations, {
        body: { integrations },
    }).as('getImageIntegrations');
    cy.intercept('GET', api.integrations.notifiers).as('getNotifierIntegrations');
    cy.intercept('GET', api.integrationHealth.externalBackups).as('getBackupIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.imageIntegrations, {
        body: { integrationHealth },
    }).as('getImageIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.notifiers).as('getNotifierIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.vulnDefinitions).as('getVulnDefinitionsHealth');

    visit(systemHealthUrl);

    cy.wait([
        '@getClusters',
        '@getBackupIntegrations',
        '@getImageIntegrations',
        '@getNotifierIntegrations',
        '@getBackupIntegrationsHealth',
        '@getImageIntegrationsHealth',
        '@getNotifierIntegrationsHealth',
        '@getVulnDefinitionsHealth',
    ]);
    cy.get('h1:contains("System Health")');
}

export function visitSystemHealthWithNotifierIntegrations(notifiers, integrationHealth) {
    cy.intercept('GET', api.clusters.list).as('getClusters');
    cy.intercept('GET', api.integrations.externalBackups).as('getBackupIntegrations');
    cy.intercept('GET', api.integrations.imageIntegrations).as('getImageIntegrations');
    cy.intercept('GET', api.integrations.notifiers, {
        body: { notifiers },
    }).as('getNotifierIntegrations');
    cy.intercept('GET', api.integrationHealth.externalBackups).as('getBackupIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.imageIntegrations).as('getImageIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.notifiers, {
        body: { integrationHealth },
    }).as('getNotifierIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.vulnDefinitions).as('getVulnDefinitionsHealth');

    visit(systemHealthUrl);

    cy.wait([
        '@getClusters',
        '@getBackupIntegrations',
        '@getImageIntegrations',
        '@getNotifierIntegrations',
        '@getBackupIntegrationsHealth',
        '@getImageIntegrationsHealth',
        '@getNotifierIntegrationsHealth',
        '@getVulnDefinitionsHealth',
    ]);
    cy.get('h1:contains("System Health")');
}

// visit vulnerability definitions

export function visitSystemHealthWithVulnerabilityDefinitionsTimestamp(lastUpdatedTimestamp) {
    cy.intercept('GET', api.clusters.list).as('getClusters');
    cy.intercept('GET', api.integrations.externalBackups).as('getBackupIntegrations');
    cy.intercept('GET', api.integrations.imageIntegrations).as('getImageIntegrations');
    cy.intercept('GET', api.integrations.notifiers).as('getNotifierIntegrations');
    cy.intercept('GET', api.integrationHealth.externalBackups).as('getBackupIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.imageIntegrations).as('getImageIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.notifiers).as('getNotifierIntegrationsHealth');
    cy.intercept('GET', api.integrationHealth.vulnDefinitions, {
        body: { lastUpdatedTimestamp },
    }).as('getVulnDefinitionsHealth');

    visit(systemHealthUrl);

    cy.wait([
        '@getClusters',
        '@getBackupIntegrations',
        '@getImageIntegrations',
        '@getNotifierIntegrations',
        '@getBackupIntegrationsHealth',
        '@getImageIntegrationsHealth',
        '@getNotifierIntegrationsHealth',
        '@getVulnDefinitionsHealth',
    ]);
    cy.get('h1:contains("System Health")');
}
