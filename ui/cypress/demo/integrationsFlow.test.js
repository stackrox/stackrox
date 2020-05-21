import {
    url as integrationsURL,
    selectors as integrationSelectors,
} from '../constants/IntegrationsPage';
import selectors from '../selectors/index';
import withAuth from '../helpers/basicAuth';

describe('Platform Configuration > Integrations Flow', () => {
    withAuth();

    it('should be able to navigate to integrations page through the left side navbar', () => {
        cy.visit('/');
        cy.get(selectors.navigation.leftNavBar, { timeout: 7000 })
            .contains('Platform Configuration')
            .click();
        cy.get(selectors.navigation.navPanel).contains('Integrations').click();
        cy.get(selectors.page.pageHeader).contains('Integrations');
    });

    it('should validate that the Slack integration is there', () => {
        cy.visit(integrationsURL);
        cy.get(integrationSelectors.tiles, { timeout: 7000 }).contains('Slack').click();
        cy.get(selectors.table.rows).eq(0);
    });

    it('should show you the form for creating a new Slack integration', () => {
        cy.visit(integrationsURL);
        cy.get(integrationSelectors.tiles, { timeout: 7000 }).contains('Slack').click();
        cy.get('button').contains('New Integration').click();
        cy.get(selectors.panel.panelHeader).eq(1).contains('New Integration');
    });
});
