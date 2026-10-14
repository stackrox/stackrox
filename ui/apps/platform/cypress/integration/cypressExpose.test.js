import { hasFeatureFlag, hasOrchestratorFlavor } from '../helpers/features';

describe('Cypress.expose public config', () => {
    it('exposes AXE_CORE_PATH from setupNodeEvents', () => {
        expect(Cypress.expose('AXE_CORE_PATH')).to.match(/axe\.min\.js$/);
    });

    it('does not expose secrets to the browser', () => {
        // Assert on key names only, so a failure never prints a secret value into the CI log.
        const exposedKeys = Object.keys(Cypress.expose());
        [
            'ROX_AUTH_TOKEN',
            'OPENSHIFT_CONSOLE_USERNAME',
            'OPENSHIFT_CONSOLE_PASSWORD',
            'DEMO_PASSWORD',
        ].forEach((secretKey) => {
            expect(exposedKeys).to.not.include(secretKey);
        });
    });

    it('copies feature flags exported by scripts/cypress.sh', function () {
        // scripts/cypress.sh always exports ROX_AUTH_TOKEN alongside the feature flags, so use it to
        // detect a runner launch. If the copy to expose breaks, hasFeatureFlag returns false and
        // flag-gated tests skip silently, so fail loudly here instead.
        cy.env(['ROX_AUTH_TOKEN']).then(({ ROX_AUTH_TOKEN }) => {
            if (!ROX_AUTH_TOKEN) {
                this.skip();
            }

            const exposedFlags = Object.entries(Cypress.expose()).filter(([key]) =>
                key.startsWith('ROX_')
            );
            expect(exposedFlags, 'ROX_* feature flags in Cypress.expose()').to.not.be.empty;
            exposedFlags.forEach(([key, value]) => {
                // Compare the type, not the value, so an unexpected secret never reaches the CI log.
                expect(typeof value, key).to.equal('boolean');
            });
        });
    });

    it('keeps secrets available to cy.env()', function () {
        cy.env(['ROX_AUTH_TOKEN']).then(({ ROX_AUTH_TOKEN }) => {
            if (!ROX_AUTH_TOKEN) {
                this.skip();
            }
            expect(ROX_AUTH_TOKEN).to.be.a('string');
        });
    });

    it('rejects the deprecated Cypress.env() because allowCypressEnv is false', () => {
        expect(() => Cypress.env('AXE_CORE_PATH')).to.throw();
    });

    it('feature helpers read from Cypress.expose', () => {
        Cypress.expose({
            ROX_FAKE_FLAG: true,
            ROX_OTHER_FLAG: false,
            ORCHESTRATOR_FLAVOR: 'openshift',
        });

        expect(hasFeatureFlag('ROX_FAKE_FLAG')).to.equal(true);
        expect(hasFeatureFlag('ROX_OTHER_FLAG')).to.equal(false);
        expect(hasFeatureFlag('ROX_MISSING_FLAG')).to.equal(false);
        expect(hasOrchestratorFlavor('openshift')).to.equal(true);
        expect(hasOrchestratorFlavor('k8s')).to.equal(false);
    });

    it('injects axe-core from the exposed AXE_CORE_PATH', () => {
        // Inject directly rather than through checkAccessibility so the assertion covers the
        // exposed path itself and does not depend on the accessibility of the current page.
        cy.injectAxe({ axeCorePath: Cypress.expose('AXE_CORE_PATH') });
        cy.window().its('axe').should('exist');
    });
});
