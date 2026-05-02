// Smoke test for CI: verifies StackRox Central serves the UI and core pages load.
// Designed to run against a minimal KinD deployment (no scanner, no collector).

describe('CI Smoke Test', () => {
    const password = Cypress.env('ROX_PASSWORD') || 'admin';

    beforeEach(() => {
        cy.login(password);
    });

    it('Dashboard loads', () => {
        cy.visit('/main/dashboard');
        cy.get('h1, [data-testid="header-text"]', { timeout: 30000 }).should('exist');
    });

    it('Clusters page shows a healthy cluster', () => {
        cy.visit('/main/clusters');
        cy.get('table, [data-testid="clusters-table"]', { timeout: 30000 }).should('exist');
    });

    it('Violations page loads', () => {
        cy.visit('/main/violations');
        cy.get('body', { timeout: 30000 }).should('contain.text', 'Violations');
    });

    it('System Health page loads', () => {
        cy.visit('/main/system-health');
        cy.get('body', { timeout: 30000 }).should('exist');
    });

    it('Risk page loads', () => {
        cy.visit('/main/risk');
        cy.get('body', { timeout: 30000 }).should('exist');
    });
});

Cypress.Commands.add('login', (password) => {
    cy.session('admin', () => {
        cy.request({
            method: 'POST',
            url: '/v1/auth/m/login',
            body: {
                username: 'admin',
                password: password,
            },
            failOnStatusCode: false,
        }).then((resp) => {
            if (resp.status === 200 && resp.body.token) {
                window.localStorage.setItem('access_token', resp.body.token);
            }
        });
    });
});
