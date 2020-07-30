import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';

describe('Policy Detail View Page', () => {
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
