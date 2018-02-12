import { url as integrationsUrl, selectors } from './pages/IntegrationsPage';

describe('Integrations page', () => {
    beforeEach(() => {
        cy.visit(integrationsUrl);
    });

    it('should have selected item in nav bar', () => {
        cy.get(selectors.navLink).should('have.class', 'bg-primary-600');
    });

    it('should allow integration with Slack', () => {
        cy.get('div.ReactModalPortal').should('not.exist');

        cy.get('button:contains("Slack")').click();
        cy.get('div.ReactModalPortal');
    });
});
