// Smoke test for CI: verifies StackRox Central serves the UI and core pages load.
// Designed to run against a minimal KinD deployment (no scanner, no collector).

describe('CI Smoke Test', () => {
    const password = Cypress.env('ROX_PASSWORD') || 'admin';

    beforeEach(() => {
        cy.request({
            method: 'POST',
            url: '/v1/auth/m/login',
            body: { username: 'admin', password },
            failOnStatusCode: false,
        }).then((resp) => {
            expect(resp.status).to.eq(200);
            window.localStorage.setItem('access_token', resp.body.token);
        });
    });

    it('API metadata endpoint responds', () => {
        cy.request('/v1/metadata').then((resp) => {
            expect(resp.status).to.eq(200);
            expect(resp.body).to.have.property('licenseStatus', 'VALID');
        });
    });

    it('Dashboard page loads', () => {
        cy.visit('/main/dashboard');
        cy.get('nav', { timeout: 30000 }).should('exist');
        cy.get('body').should('not.contain.text', 'Log in to your account');
    });

    it('Violations page loads', () => {
        cy.visit('/main/violations');
        cy.get('nav', { timeout: 30000 }).should('exist');
        cy.get('body').should('not.contain.text', 'Log in to your account');
    });

    it('System Health page loads', () => {
        cy.visit('/main/system-health');
        cy.get('nav', { timeout: 30000 }).should('exist');
    });

    it('Risk page loads', () => {
        cy.visit('/main/risk');
        cy.get('nav', { timeout: 30000 }).should('exist');
    });
});
