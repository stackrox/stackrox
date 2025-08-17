export function withOcpAuth() {
    if (Cypress.env('OCP_BRIDGE_AUTH_DISABLED')) {
        return;
    }

    cy.session('ocp-session-auth', () => {
        cy.visit('/', { timeout: 60000 });
        cy.get('input[name="username"]').type(Cypress.env('CLUSTER_USERNAME'));
        cy.get('input[name="password"]').type(Cypress.env('CLUSTER_PASSWORD'));
        cy.get('button[type="submit"]').click();
        // TODO Handle OCP welcome modal
    });
}
