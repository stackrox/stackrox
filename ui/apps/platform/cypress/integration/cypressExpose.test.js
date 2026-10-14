import { hasFeatureFlag, hasOrchestratorFlavor } from '../helpers/features';

// Assertions report key names only, so a failure never prints a secret value into the CI log.
const SECRET_KEYS = [
    'ROX_AUTH_TOKEN',
    'OPENSHIFT_CONSOLE_USERNAME',
    'OPENSHIFT_CONSOLE_PASSWORD',
    'DEMO_PASSWORD',
];

describe('Cypress.expose public config', () => {
    it('does not expose secrets to the browser', () => {
        const exposed = Cypress.expose();
        const leakedKeys = SECRET_KEYS.filter((key) => key in exposed);
        // Feature flags are always booleans, so any other ROX_* value is an unexpected leak.
        const nonBooleanRoxKeys = Object.keys(exposed).filter(
            (key) => key.startsWith('ROX_') && typeof exposed[key] !== 'boolean'
        );

        expect(leakedKeys, 'secrets in Cypress.expose()').to.be.empty;
        expect(nonBooleanRoxKeys, 'non-boolean ROX_* values in Cypress.expose()').to.be.empty;
    });

    it('copies feature flags exported by scripts/cypress.sh as booleans', function () {
        // scripts/cypress.sh always exports ROX_AUTH_TOKEN alongside the feature flags, so use it to
        // detect a runner launch. If the copy breaks, flag-gated tests skip silently, so fail here.
        cy.env(['ROX_AUTH_TOKEN']).then(({ ROX_AUTH_TOKEN }) => {
            if (!ROX_AUTH_TOKEN) {
                this.skip();
            }

            const flagKeys = Object.keys(Cypress.expose()).filter((key) => key.startsWith('ROX_'));

            expect(flagKeys, 'ROX_* feature flags in Cypress.expose()').to.not.be.empty;
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
