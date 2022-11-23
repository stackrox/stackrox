import * as api from '../../constants/apiEndpoints';
import { url as dashboardUrl } from '../../constants/DashboardPage';
import { labels, selectors, url } from '../../constants/IntegrationsPage';
import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

function getIntegrationTypeUrl(integrationSource, integrationType) {
    return `${url}/${integrationSource}/${integrationType}`;
}

function visitIntegrations() {
    cy.intercept('GET', api.roles.mypermissions).as('getMyPermissions'); // for left nav
    cy.intercept('GET', api.search.options).as('getSearchOptions'); // near the end of the requests
    cy.visit(dashboardUrl);
    cy.wait(['@getMyPermissions', '@getSearchOptions']);

    cy.intercept('GET', api.integrations.apiTokens).as('getApiTokens');
    cy.intercept('GET', api.integrations.clusterInitBundles).as('getClusterInitBundles');
    cy.intercept('GET', api.integrations.externalBackups).as('getExternalBackups');
    cy.intercept('GET', api.integrations.imageIntegrations).as('getImageIntegrations');
    cy.intercept('GET', api.integrations.notifiers).as('getNotifiers');
    cy.get(selectors.configure).click();
    cy.get(selectors.navLink).click();
    cy.wait([
        '@getApiTokens',
        '@getClusterInitBundles',
        '@getExternalBackups',
        '@getImageIntegrations',
        '@getNotifiers',
    ]);
    cy.get(`${selectors.title1}:contains("Integrations")`);
}

describe('Integrations page', () => {
    withAuth();

    it('should have title', () => {
        visitIntegrations();

        cy.title().should('match', getRegExpForTitleWithBranding('Integrations'));
    });

    it('Plugin tiles should all be the same height', () => {
        visitIntegrations();

        let value = null;
        cy.get(selectors.plugins).each(($el) => {
            if (value) {
                expect($el[0].clientHeight).to.equal(value);
            } else {
                value = $el[0].clientHeight;
            }
        });
    });

    it('should go to the table for a type of imageIntegrations', () => {
        const integrationSourceLabel = 'Image Integrations'; // TODO might change from Title Case to Sentence case
        const integrationTypeLabel = labels.imageIntegrations.docker;
        const integrationTypeUrl = getIntegrationTypeUrl('imageIntegrations', 'docker');

        visitIntegrations();

        cy.get(`${selectors.title2}:contains("${integrationSourceLabel}")`);
        cy.get(`${selectors.tile}:contains("${integrationTypeLabel}")`).click();

        // Verify that tests for a type of the source can visit directly.
        cy.location('pathname').should('eq', integrationTypeUrl);

        cy.get(`${selectors.breadcrumbItem}:contains("${integrationTypeLabel}")`);
        cy.get(`${selectors.title2}:contains("${integrationTypeLabel}")`);
        cy.get(selectors.buttons.newIntegration);
    });

    it('should go to the table for a type of notifiers', () => {
        const integrationSourceLabel = 'Notifier Integrations'; // TODO might change from Title Case to Sentence case
        const integrationTypeLabel = labels.notifiers.slack;
        const integrationTypeUrl = getIntegrationTypeUrl('notifiers', 'slack');

        visitIntegrations();

        cy.get(`${selectors.title2}:contains("${integrationSourceLabel}")`);
        cy.get(`${selectors.tile}:contains("${integrationTypeLabel}")`).click();

        // Verify that tests for a type of the source can visit directly.
        cy.location('pathname').should('eq', integrationTypeUrl);

        cy.get(`${selectors.breadcrumbItem}:contains("${integrationTypeLabel}")`);
        cy.get(`${selectors.title2}:contains("${integrationTypeLabel}")`);
        cy.get(selectors.buttons.newIntegration);
    });

    it('should go to the table for a type of backups', () => {
        const integrationSourceLabel = 'Backup Integrations'; // TODO might change from Title Case to Sentence case
        const integrationTypeLabel = labels.backups.s3;
        const integrationTypeUrl = getIntegrationTypeUrl('backups', 's3');

        visitIntegrations();

        cy.get(`${selectors.title2}:contains("${integrationSourceLabel}")`);
        cy.get(`${selectors.tile}:contains("${integrationTypeLabel}")`).click();

        // Verify that tests for a type of the source can visit directly.
        cy.location('pathname').should('eq', integrationTypeUrl);

        cy.get(`${selectors.breadcrumbItem}:contains("${integrationTypeLabel}")`);
        cy.get(`${selectors.title2}:contains("${integrationTypeLabel}")`);
        cy.get(selectors.buttons.newIntegration);
    });

    it('should go to the table for apitoken type of authProviders', () => {
        const integrationSourceLabel = 'Authentication Tokens'; // TODO might change from Title Case to Sentence case
        const integrationTypeLabel = labels.authProviders.apitoken;
        const integrationTypeUrl = getIntegrationTypeUrl('authProviders', 'apitoken');

        visitIntegrations();

        cy.get(`${selectors.title2}:contains("${integrationSourceLabel}")`);
        cy.get(`${selectors.tile}:contains("${integrationTypeLabel}")`).click();

        // Verify that tests for a type of the source can visit directly.
        cy.location('pathname').should('eq', integrationTypeUrl);

        cy.get(`${selectors.breadcrumbItem}:contains("${integrationTypeLabel}")`);
        cy.get(`${selectors.title2}:contains("${integrationTypeLabel}")`);
        cy.get(selectors.buttons.newApiToken);
    });

    it('should go to the table for clusterInitBundle type of authProviders', () => {
        const integrationSourceLabel = 'Authentication Tokens'; // TODO might change from Title Case to Sentence case
        const integrationTypeLabel = labels.authProviders.clusterInitBundle;
        const integrationTypeUrl = getIntegrationTypeUrl('authProviders', 'clusterInitBundle');

        visitIntegrations();

        cy.get(`${selectors.title2}:contains("${integrationSourceLabel}")`);
        cy.get(`${selectors.tile}:contains("${integrationTypeLabel}")`).click();

        // Verify that tests for a type of the source can visit directly.
        cy.location('pathname').should('eq', integrationTypeUrl);

        cy.get(`${selectors.breadcrumbItem}:contains("${integrationTypeLabel}")`);
        cy.get(`${selectors.title2}:contains("${integrationTypeLabel}")`);
        cy.get(selectors.buttons.newClusterInitBundle).click();
    });
});
