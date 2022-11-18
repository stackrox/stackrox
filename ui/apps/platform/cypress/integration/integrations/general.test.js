import { labels, selectors } from '../../constants/IntegrationsPage';
import withAuth from '../../helpers/basicAuth';
import {
    assertIntegrationsTable,
    visitIntegrationsDashboard,
    visitIntegrationsDashboardFromLeftNav,
} from '../../helpers/integrations';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

describe('Integrations Dashboard', () => {
    withAuth();

    it('should visit via link in left nav', () => {
        visitIntegrationsDashboardFromLeftNav();
    });

    it('should have title', () => {
        visitIntegrationsDashboard();

        cy.title().should('match', getRegExpForTitleWithBranding('Integrations'));
    });

    it('Plugin tiles should all be the same height', () => {
        visitIntegrationsDashboard();

        let value = null;
        cy.get(selectors.plugins).each(($el) => {
            if (value) {
                expect($el[0].clientHeight).to.equal(value);
            } else {
                value = $el[0].clientHeight;
            }
        });
    });

    // Page address segments are the source of truth for integrationSource and integrationType.

    it('should go to the table for a type of imageIntegrations', () => {
        const integrationSource = 'imageIntegrations';
        const integrationSourceLabel = 'Image Integrations'; // TODO might change from Title Case to Sentence case

        const integrationType = 'docker';
        const integrationTypeLabel = labels[integrationSource][integrationType];

        visitIntegrationsDashboard();

        cy.get(`${selectors.title2}:contains("${integrationSourceLabel}")`);
        cy.get(`${selectors.tile}:contains("${integrationTypeLabel}")`).click();

        assertIntegrationsTable(integrationSource, integrationType);
    });

    it('should go to the table for a type of notifiers', () => {
        const integrationSource = 'notifiers';
        const integrationSourceLabel = 'Notifier Integrations'; // TODO might change from Title Case to Sentence case

        const integrationType = 'slack';
        const integrationTypeLabel = labels[integrationSource][integrationType];

        visitIntegrationsDashboard();

        cy.get(`${selectors.title2}:contains("${integrationSourceLabel}")`);
        cy.get(`${selectors.tile}:contains("${integrationTypeLabel}")`).click();

        assertIntegrationsTable(integrationSource, integrationType);
    });

    it('should go to the table for a type of backups', () => {
        const integrationSource = 'backups';
        const integrationSourceLabel = 'Backup Integrations'; // TODO might change from Title Case to Sentence case

        const integrationType = 's3';
        const integrationTypeLabel = labels[integrationSource][integrationType];

        visitIntegrationsDashboard();

        cy.get(`${selectors.title2}:contains("${integrationSourceLabel}")`);
        cy.get(`${selectors.tile}:contains("${integrationTypeLabel}")`).click();

        assertIntegrationsTable(integrationSource, integrationType);
    });

    it('should go to the table for apitoken type of authProviders', () => {
        const integrationSource = 'authProviders';
        const integrationSourceLabel = 'Authentication Tokens'; // TODO might change from Title Case to Sentence case

        const integrationType = 'apitoken';
        const integrationTypeLabel = labels[integrationSource][integrationType];

        visitIntegrationsDashboard();

        cy.get(`${selectors.title2}:contains("${integrationSourceLabel}")`);
        cy.get(`${selectors.tile}:contains("${integrationTypeLabel}")`).click();

        assertIntegrationsTable(integrationSource, integrationType);
    });

    it('should go to the table for clusterInitBundle type of authProviders', () => {
        const integrationSource = 'authProviders';
        const integrationSourceLabel = 'Authentication Tokens'; // TODO might change from Title Case to Sentence case

        const integrationType = 'clusterInitBundle';
        const integrationTypeLabel = labels[integrationSource][integrationType];

        visitIntegrationsDashboard();

        cy.get(`${selectors.title2}:contains("${integrationSourceLabel}")`);
        cy.get(`${selectors.tile}:contains("${integrationTypeLabel}")`).click();

        assertIntegrationsTable(integrationSource, integrationType);
    });
});
