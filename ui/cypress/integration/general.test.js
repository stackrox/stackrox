import { url as dashboardUrl } from './constants/DashboardPage';
import selectors from './constants/GeneralPage';

//
// Sanity / general checks for UI being up and running
//

describe('General sanity checks', () => {
    it('should have correct <title>', () => {
        cy.visit('/');
        cy.title().should('eq', 'StackRox');
    });

    it('should render navbar with Dashboard selected', () => {
        cy.visit('/');
        cy.get(selectors.navLinks.first).as('firstNavItem');
        cy.get(selectors.navLinks.others).as('otherNavItems');

        // redirect should happen
        cy.url().should('contain', dashboardUrl);

        // Dashboard is selected
        cy.get('@firstNavItem').should('have.class', 'bg-primary-700');
        cy.get('@firstNavItem').contains('Dashboard');

        // nothing else is selected
        cy.get('@otherNavItems').should('not.have.class', 'bg-primary-700');

        cy.get(selectors.navLinks.list).as('topNavItems');
        cy.get('@topNavItems').should($lis => {
            expect($lis).to.have.length(6);
            expect($lis.eq(0)).to.contain('Cluster');
            expect($lis.eq(1)).to.contain('Node');
            expect($lis.eq(2)).to.contain('Violation');
            expect($lis.eq(3)).to.contain('Deployment');
            expect($lis.eq(4)).to.contain('Image');
            expect($lis.eq(5)).to.contain('Secret');
        });
    });

    it('should handle toggle click on Compliance navigation link', () => {
        cy.visit('/');

        cy.get(selectors.navLinks.compliance).as('compliance');

        cy.get('@compliance').click();

        cy.get(selectors.sidePanel).should('be.visible');

        cy.get('@compliance').click();

        cy.get('.navigation-panel').should('not.be.visible');
    });

    it('should go to API docs', () => {
        cy.visit('/');
        cy.get(selectors.navLinks.apidocs).as('apidocs');
        cy.get('@apidocs').click();
        cy.url().should('contain', 'apidocs');
    });
});
