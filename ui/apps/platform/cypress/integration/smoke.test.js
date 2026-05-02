// Smoke test for CI: verifies StackRox Central serves the UI and core pages load.
// Designed to run against a minimal KinD deployment (no scanner, no collector).
// Auth: CYPRESS_ROX_AUTH_TOKEN env var set by the workflow (via /v1/apitokens/generate).

describe('CI Smoke Test', () => {
    beforeEach(() => {
        const token = Cypress.env('ROX_AUTH_TOKEN');
        if (token) {
            localStorage.setItem('access_token', token);
        }
    });

    it('API metadata endpoint responds', () => {
        cy.request({
            url: '/v1/metadata',
            auth: { bearer: Cypress.env('ROX_AUTH_TOKEN') },
        }).then((resp) => {
            expect(resp.status).to.eq(200);
            expect(resp.body).to.have.property('licenseStatus', 'VALID');
        });
    });

    it('Dashboard page loads', () => {
        cy.visit('/main/dashboard');
        cy.get('nav', { timeout: 30000 }).should('exist');
    });

    it('Violations page loads', () => {
        cy.visit('/main/violations');
        cy.get('nav', { timeout: 30000 }).should('exist');
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
