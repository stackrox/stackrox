export function withOcpAuth() {
    // Establish a cookie based session for the OCP web console
    cy.session('ocp-session-auth', () => {
        cy.visit('/');

        if (!Cypress.env('OCP_BRIDGE_AUTH_DISABLED')) {
            cy.url().should('contain', '/login?');
            cy.get('input[name="username"]').type(Cypress.env('OPENSHIFT_CONSOLE_USERNAME'));
            cy.get('input[name="password"]').type(Cypress.env('OPENSHIFT_CONSOLE_PASSWORD'));
            cy.get('button[type="submit"]').click();
        }

        // Wait for the page to load
        cy.url().should('contain', '/dashboards');
        cy.get('h1:contains("Overview")');

        // Pressing Escape closes the welcome modal if it exists, and silently does nothing if it doesn't
        cy.get('body').type('{esc}');
    });
}
