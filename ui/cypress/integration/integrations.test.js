describe('Integrations page', () => {
    beforeEach(() => {
        cy.visit('/main/integrations');
    });

    it('should have selected item in nav bar', () => {
        cy.get('nav li:contains("Integrations") a').should('have.class', 'bg-primary-600');
    });

    it('should allow integration with Slack', () => {
        cy.get('div.ReactModalPortal').should('not.exist');

        cy.get('button:contains("Slack")').click();
        cy.get('div.ReactModalPortal');
    });
});
