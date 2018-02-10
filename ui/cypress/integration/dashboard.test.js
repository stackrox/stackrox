describe('Dashboard page', () => {
    beforeEach(() => {
        cy.visit('/main/dashboard');
    });

    it('should select item in nav bar', () => {
        cy.get('nav li:contains("Dashboard") a').should('have.class', 'bg-primary-600');
    });

    it('should have violations by cluster chart', () => {
        cy.get('h2:contains("Violations by Cluster")').next().within(() => {
            cy.get('svg.recharts-surface');
        });
    });

    it('should have events by time charts', () => {
        cy.get('h2:contains("Events by Time")').next().within(() => {
            cy.get('svg.recharts-surface');
        });
    });
});
