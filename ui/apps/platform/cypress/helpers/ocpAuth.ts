export function withOcpAuth() {
    if (Cypress.env('OCP_BRIDGE_AUTH_DISABLED')) {
        return;
    }

    cy.session('ocp-session-auth', () => {
        cy.visit('/', { timeout: 6000 });
        cy.url().should("contain", "/login?");
        // 8s timeout for get?
        cy.get('input[name="username"]').type(Cypress.env('CLUSTER_USERNAME'));
        cy.get('input[name="password"]').type(Cypress.env('CLUSTER_PASSWORD'));
        cy.get('button[type="submit"]').click();
        cy.url().should("contain", "/dashboards");
        cy.contains('Skip tour', { timeout: 10000 }).click()
        cy.wait(1000)
        // TODO Handle OCP welcome modal
    });
}
