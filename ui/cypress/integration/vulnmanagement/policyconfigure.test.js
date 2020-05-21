import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import checkFeatureFlag from '../../helpers/features';

describe('Policy Detail View Page', () => {
    before(function beforeHook() {
        // skip the whole suite if vuln mgmt isn't enabled
        if (checkFeatureFlag('ROX_VULN_MGMT_UI', false)) {
            this.skip();
        }
    });

    withAuth();

    it('policy edit button on click goes to edit page of policy ', () => {
        cy.visit(url.list.policies);
        cy.get(selectors.tableBodyColumn)
            .eq(0)
            .invoke('text')
            .then((value) => {
                cy.get(selectors.tableBodyColumn).eq(0).click({ force: true });
                cy.get(selectors.policyEditButton).click({ force: true });
                cy.url().then((policyEditURL) => {
                    expect(policyEditURL).to.include(`/edit`);
                });
                cy.url().then((policyEditURL) => {
                    expect(policyEditURL).to.include(value);
                });
            });
    });
});
