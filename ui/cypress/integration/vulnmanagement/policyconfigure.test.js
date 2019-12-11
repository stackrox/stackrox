import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import checkFeatureFlag from '../../helpers/features';

function getPolicyID(editUrl) {
    const res = editUrl.split('=');
    return res[2];
}
describe('Policy Detail View Page', () => {
    before(function beforeHook() {
        // skip the whole suite if vuln mgmt isn't enabled
        if (checkFeatureFlag('ROX_VULN_MGMT_UI', false)) {
            this.skip();
        }
    });

    withAuth();

    it('enable disable toggle for a policy work as expected', () => {
        cy.visit(url.list.policies);
        cy.get(selectors.tableFirstColumn)
            .eq(0)
            .click({ force: true });
        cy.url().then(currentURL => {
            const policyId = getPolicyID(currentURL);
            cy.get(selectors.policyEditButton).click({ force: true });
            cy.url().then(policyEditURL => {
                expect(policyEditURL).to.contains(`policies/${policyId}/edit`);
            });
        });
    });
});
