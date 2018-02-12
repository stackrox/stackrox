import { url as dashboardUrl } from './pages/DashboardPage';

//
// Sanity / general checks for UI being up and running
//

describe('General sanity checks', () => {
    it('should have correct <title>', () => {
        cy.visit('/');
        cy.title().should('eq', 'Mitigate');
    });

    it('should render navbar with Dashboard selected', () => {
        cy.visit('/');
        cy.get('nav li:first a').as('firstNavItem');
        cy.get('nav li:not(:first) a').as('otherNavItems');

        // redirect should happen
        cy.url().should('contain', dashboardUrl);

        // Dashboard is selected
        cy.get('@firstNavItem').should('have.class', 'bg-primary-600');
        cy.get('@firstNavItem').contains('Dashboard');

        // nothing else is selected
        cy.get('@otherNavItems').should('not.have.class', 'bg-primary-600');
    });
});
