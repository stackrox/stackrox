import { url as violationsUrl, selectors as ViolationsPageSelectors } from './pages/ViolationsPage';
import selectors from './pages/SearchPage';

describe('Violations page', () => {
    it('should select item in nav bar', () => {
        cy.visit(violationsUrl);
        cy.get(ViolationsPageSelectors.navLink).should('have.class', 'bg-primary-600');
    });

    it('should close the side panel on search filter', () => {
        cy.get(selectors.searchInput).type('Cluster:{enter}', { force: true });
        cy.get(selectors.searchInput).type('remote{enter}', { force: true });
        cy.get('.side-panel').should('not.be.visible');
    });
});
