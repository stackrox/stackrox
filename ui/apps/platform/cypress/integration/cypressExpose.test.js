import { hasFeatureFlag, hasOrchestratorFlavor } from '../helpers/features';

describe('Cypress.expose public config', () => {
    it('exposes AXE_CORE_PATH from setupNodeEvents', () => {
        expect(Cypress.expose('AXE_CORE_PATH')).to.match(/axe\.min\.js$/);
    });

    it('does not expose secrets to the browser', () => {
        expect(Cypress.expose('ROX_AUTH_TOKEN')).to.equal(undefined);
        expect(Cypress.expose('OPENSHIFT_CONSOLE_USERNAME')).to.equal(undefined);
        expect(Cypress.expose('OPENSHIFT_CONSOLE_PASSWORD')).to.equal(undefined);
        expect(Object.keys(Cypress.expose())).to.not.include.members([
            'ROX_AUTH_TOKEN',
            'OPENSHIFT_CONSOLE_USERNAME',
            'OPENSHIFT_CONSOLE_PASSWORD',
        ]);
    });

    it('copies CYPRESS_* public values when the runner exports them', function () {
        if (Cypress.expose('ROX_FAKE_FLAG') === undefined) {
            this.skip();
        }

        expect(Cypress.expose('ROX_FAKE_FLAG')).to.equal(true);
        expect(Cypress.expose('ROX_OTHER_FLAG')).to.equal(false);
        expect(Cypress.expose('ORCHESTRATOR_FLAVOR')).to.equal('openshift');
    });

    it('keeps secrets available to cy.env()', function () {
        cy.env(['ROX_AUTH_TOKEN']).then(({ ROX_AUTH_TOKEN }) => {
            if (!ROX_AUTH_TOKEN) {
                this.skip();
            }
            expect(ROX_AUTH_TOKEN).to.be.a('string');
            expect(Cypress.expose('ROX_AUTH_TOKEN')).to.equal(undefined);
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
