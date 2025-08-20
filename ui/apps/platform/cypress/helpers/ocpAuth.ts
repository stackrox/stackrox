export function withOcpAuth() {
    if (Cypress.env('OCP_BRIDGE_AUTH_DISABLED')) {
        return;
    }

    cy.session('ocp-session-auth', () => {
        cy.visit('/');
        cy.url().should('contain', '/login?');
        cy.get('input[name="username"]').type(Cypress.env('OPENSHIFT_CONSOLE_USERNAME'));
        cy.get('input[name="password"]').type(Cypress.env('OPENSHIFT_CONSOLE_PASSWORD'));
        cy.get('button[type="submit"]').click();
        cy.url().should('contain', '/dashboards');
        cy.contains('Skip tour', { timeout: 10000 }).click();
        cy.wait(1000);
        // TODO Handle OCP welcome modal
    });
}
