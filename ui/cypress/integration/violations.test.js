import { url as violationsUrl, selectors } from './pages/ViolationsPage';

describe('Violations page', () => {
    it('should select item in nav bar', () => {
        cy.visit(violationsUrl);
        cy.get(selectors.navLink).should('have.class', 'bg-primary-600');
    });
});
