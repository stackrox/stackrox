describe('Compliance page', () => {
    beforeEach(() => {
        cy.visit('/main/compliance');
    });

    it('should have selected item in nav bar', () => {
        cy.get('nav li:contains("Compliance") a').should('have.class', 'bg-primary-600');
    });

    it('should start scanning', () => {
        // first tab selected by default
        cy.get('button.tab:first').should('have.class', 'tab-active');

        cy.get('button:contains("Scan now")').as('scanNow');

        // start scanning
        cy
            .get('@scanNow')
            .children()
            .should('have.length', 0);
        cy.get('@scanNow').click();
        cy
            .get('@scanNow')
            .children()
            .should('have.length', 1); // spinner should appear
    });
});
