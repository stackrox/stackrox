export function withOcpAuth() {
    // Establish a cookie based session for the OCP web console
    cy.session('ocp-session-auth', () => {
        cy.env([
            'OCP_BRIDGE_AUTH_DISABLED',
            'OPENSHIFT_CONSOLE_USERNAME',
            'OPENSHIFT_CONSOLE_PASSWORD',
        ]).then(
            ({
                OCP_BRIDGE_AUTH_DISABLED,
                OPENSHIFT_CONSOLE_USERNAME,
                OPENSHIFT_CONSOLE_PASSWORD,
            }) => {
                cy.visit('/');

                if (!OCP_BRIDGE_AUTH_DISABLED) {
                    cy.url().should('contain', '/login?');
                    cy.get('input[name="username"]').type(OPENSHIFT_CONSOLE_USERNAME);
                    cy.get('input[name="password"]').type(OPENSHIFT_CONSOLE_PASSWORD, {
                        log: false,
                    });
                    cy.get('button[type="submit"]').click();
                }

                // Wait for the page to load
                cy.url().should('contain', '/dashboards');
                cy.get('h1:contains("Overview")');

                // Pressing Escape closes the welcome modal if it exists, and silently does nothing if it doesn't
                cy.get('body').type('{esc}');
            }
        );
    });
}
