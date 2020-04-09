import withAuth from '../helpers/basicAuth';

describe('Risk page', () => {
    withAuth();

    it('main menu bar should not have images item', () => {
        cy.visit('/');
        cy.get('nav.left-navigation li:contains("Images") a').should('not.exist');
    });
});
