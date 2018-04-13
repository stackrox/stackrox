import { url as dashboardUrl } from './pages/DashboardPage';

//
// Sanity / general checks for UI being up and running
//

describe('General sanity checks', () => {
    it('should have correct <title>', () => {
        cy.visit('/');
        cy.title().should('eq', 'Prevent');
    });

    it('should render navbar with Dashboard selected', () => {
        cy.visit('/');
        cy.get('nav.left-navigation li:first a').as('firstNavItem');
        cy.get('nav.left-navigation li:not(:first) a').as('otherNavItems');

        // redirect should happen
        cy.url().should('contain', dashboardUrl);

        // Dashboard is selected
        cy.get('@firstNavItem').should('have.class', 'bg-primary-600');
        cy.get('@firstNavItem').contains('Dashboard');

        // nothing else is selected
        cy.get('@otherNavItems').should('not.have.class', 'bg-primary-600');

        cy.get('nav.top-navigation li').as('topNavItems');
        cy.get('@topNavItems').should($lis => {
            expect($lis).to.have.length(4);
            expect($lis.eq(0)).to.contain('Violations');
            expect($lis.eq(1)).to.contain('Cluster');
            expect($lis.eq(2)).to.contain('Deployments');
            expect($lis.eq(3)).to.contain('Image');
        });
    });
});
