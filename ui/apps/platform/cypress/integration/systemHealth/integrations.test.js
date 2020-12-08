import { url, selectors } from '../../constants/SystemHealth';
import { integrations as integrationsApi } from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';

describe('System Health Integrations local deployment', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.route('GET', integrationsApi.notifiers).as('GetNotifiers');
    });

    it('should go from left navigation to Dashboard and have widgets', () => {
        cy.visit('/');
        cy.get('nav.left-navigation a:contains("Platform Configuration")').click();
        cy.get('[data-testid="configure-subnav"] a:contains("System Health")').click();
        cy.wait('@GetNotifiers');

        cy.get('[data-testid="header-text"]').should('have.text', 'System Health');

        Object.entries({
            imageIntegrations: 'Image Integrations',
            pluginIntegrations: 'Plugin Integrations',
            backupIntegrations: 'Backup Integrations',
        }).forEach(([key, text]) => {
            cy.get(`${selectors.integrations.widgets[key]} [data-testid="widget-header"]`).should(
                'have.text',
                text
            );
        });
    });

    it('should go from Dashboard to Plugins anchor on Integrations page via click View All', () => {
        cy.visit(url.dashboard);
        cy.wait('@GetNotifiers');

        cy.get(
            `${selectors.integrations.widgets.pluginIntegrations} ${selectors.integrations.viewAllButton}`
        ).click();
        cy.wait('@GetNotifiers');

        cy.get('[data-testid="header-text"]').should('have.text', 'Integrations');
        cy.get('#plugin-integrations h2:contains("Plugins")').should('be.visible');
    });
});
