import { selectors } from '../../constants/PoliciesPage';
import withAuth from '../../helpers/basicAuth';
import {
    doPolicyPageAction,
    doPolicyRowAction,
    visitPolicies,
    visitPolicy,
} from '../../helpers/policies';
import { interceptAndWatchRequests } from '../../helpers/request';

const saveAsUrl = '/v1/policies/save-as';

const routeMatcherMapForPolicySaveAs = {
    [saveAsUrl]: { method: 'POST', url: saveAsUrl },
};

const firstTableRowSelector = 'tbody:nth-child(2) tr:nth-child(1)';

function visitPoliciesAndGetFirstPolicyId() {
    visitPolicies();
    return cy.get(`${firstTableRowSelector} ${selectors.table.policyLink}`).then(($a) => {
        const segments = $a.attr('href')?.split('/') ?? [];
        return segments[segments.length - 1];
    });
}

describe('Save policies as Custom Resource', () => {
    withAuth();

    describe('policies table', () => {
        it('should save policy as Custom Resource via table row menu', () => {
            visitPoliciesAndGetFirstPolicyId().then((policyId) => {
                interceptAndWatchRequests(routeMatcherMapForPolicySaveAs)
                    .then(({ waitForRequests }) => {
                        doPolicyRowAction(firstTableRowSelector, 'Save as Custom Resource');

                        return waitForRequests();
                    })
                    .then(({ request }) => {
                        expect(request.body).to.deep.equal({ policyIds: [policyId] });
                        // TODO Expect ZIP

                        cy.get(
                            `${selectors.toast.title}:contains("Successfully saved selected policies as Custom Resource")`
                        );
                    });
            });
        });

        it('should allow export of multiple policies via the Bulk actions menu', () => {
            visitPolicies();

            cy.get(selectors.table.bulkActionsDropdownButton).should('be.disabled');
            cy.get(`tbody ${selectors.table.selectCheckbox}:eq(0)`)
                .should('not.be.checked')
                .click();
            cy.get(`tbody ${selectors.table.selectCheckbox}:eq(1)`)
                .should('not.be.checked')
                .click();
            cy.get(selectors.table.bulkActionsDropdownButton).should('be.enabled').click();

            interceptAndWatchRequests(routeMatcherMapForPolicySaveAs)
                .then(({ waitForRequests }) => {
                    cy.get(
                        `${selectors.table.bulkActionsDropdownItem}:contains("Save as Custom Resource")`
                    ).click();

                    return waitForRequests();
                })
                .then(({ request }) => {
                    // Request has policy ids.
                    expect(request.body.policyIds).to.have.length(2);
                    // TODO Expect Zip
                    cy.get(
                        `${selectors.toast.title}:contains("Successfully saved selected policies as Custom Resources")`
                    );
                });
        });

        it('should display toast alert for export service failure', () => {
            visitPolicies();

            interceptAndWatchRequests(routeMatcherMapForPolicySaveAs, {
                [saveAsUrl]: { statusCode: 400 },
            }).then(({ waitForRequests }) => {
                doPolicyRowAction(firstTableRowSelector, 'Save as Custom Resource');

                waitForRequests();

                cy.get(
                    `${selectors.toast.title}:contains("Could not save the selected policies as Custom Resource")`
                );
            });
        });
    });

    describe('policy detail page', () => {
        it('should save policy as custom resource', () => {
            visitPoliciesAndGetFirstPolicyId().then((policyId) => {
                visitPolicy(policyId);

                interceptAndWatchRequests(routeMatcherMapForPolicySaveAs)
                    .then(({ waitForRequests }) => {
                        doPolicyPageAction('Save as Custom Resource');
                        return waitForRequests();
                    })
                    .then(({ request }) => {
                        // Request has expected policy id.
                        expect(request.body).to.deep.equal({ policyIds: [policyId] });
                        // TODO Expect ZIP
                        cy.get(
                            `${selectors.toast.title}:contains("Successfully saved policy as Custom Resource")`
                        );
                    });
            });
        });

        it('should display toast alert for "save as" failure', () => {
            visitPoliciesAndGetFirstPolicyId().then((policyId) => {
                visitPolicy(policyId);

                interceptAndWatchRequests(routeMatcherMapForPolicySaveAs, {
                    [saveAsUrl]: { statusCode: 400 },
                }).then(({ waitForRequests }) => {
                    doPolicyPageAction('Save as Custom Resource');

                    waitForRequests();

                    cy.get(
                        `${selectors.toast.title}:contains("Could not save policy as Custom Resource")`
                    );
                });
            });
        });
    });
});
