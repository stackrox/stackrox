import { selectors, url } from './constants/Docs';

describe('Documentation Access', () => {
    it('should load the documentation page', () => {
        cy.visit(url);
        cy.url().should('contain', url); // Check there wasn't a redirect
        cy.get(selectors.homeBanner);
    });
});
