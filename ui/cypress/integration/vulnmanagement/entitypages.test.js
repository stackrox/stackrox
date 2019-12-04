import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import checkFeatureFlag from '../../helpers/features';

describe('Entities single views', () => {
    before(function beforeHook() {
        // skip the whole suite if vuln mgmt isn't enabled
        if (checkFeatureFlag('ROX_VULN_MGMT_UI', false)) {
            this.skip();
        }
    });

    withAuth();

    it('related entities tile links should unset search params upon navigation', () => {
        // arrange
        cy.visit(url.list.clusters);

        cy.get(selectors.tableRows)
            .eq(0)
            .get(selectors.fixableCvesLink)
            .click();

        cy.get(selectors.backButton).click();

        // act
        cy.get(selectors.tileLinks)
            .eq(1)
            .find(selectors.tileLinkSuperText)
            .invoke('text')
            .then(value => {
                const numDeployments = value;

                cy.get(selectors.tileLinks)
                    .eq(1)
                    // force: true option needed because this open issue for cypress
                    //   https://github.com/cypress-io/cypress/issues/4856
                    .click({ force: true });

                cy.get(`[data-test-id="side-panel"] [data-test-id="panel-header"]`)
                    .invoke('text')
                    .then(panelHeaderText => {
                        expect(parseInt(panelHeaderText, 10)).to.equal(
                            parseInt(numDeployments, 10)
                        );
                    });
            });

        // assert
    });
});
