import { selectors } from '../../constants/PoliciesPage';
import withAuth from '../../helpers/basicAuth';
import {
    deletePolicyIfExists,
    doPolicyPageAction,
    doPolicyRowAction,
    importPolicyFromFixture,
    visitPolicies,
    visitPolicy,
} from '../../helpers/policies';
import { interceptAndWatchRequests } from '../../helpers/request';

const saveAsUrl = '/api/policy/custom-resource/save-as-zip';

const routeMatcherMapForPolicySaveAs = {
    [saveAsUrl]: { method: 'POST', url: saveAsUrl },
};

const policyWithNameSelector = (name) => `tbody tr:has(td:contains("${name}")):eq(0)`;
const importedPolicyFixtureName = 'Severity greater than moderate';

function getPolicyIdFromRowWithName(name) {
    return cy.get(`${policyWithNameSelector(name)} ${selectors.table.policyLink}`).then(($a) => {
        const segments = $a.attr('href')?.split('/') ?? [];
        return segments[segments.length - 1];
    });
}

describe('Save policies as Custom Resource', () => {
    withAuth();

    // Clean up the two policies that will be created during the tests
    beforeEach(() => {
        deletePolicyIfExists(importedPolicyFixtureName);
        deletePolicyIfExists(`${importedPolicyFixtureName} 2`);
    });

    afterEach(() => {
        deletePolicyIfExists(importedPolicyFixtureName);
        deletePolicyIfExists(`${importedPolicyFixtureName} 2`);
    });

    describe('policies table', () => {
        it('should save policy as Custom Resource via table row menu', () => {
            visitPolicies();
            importPolicyFromFixture('policies/good_policy_to_import.json');

            getPolicyIdFromRowWithName(importedPolicyFixtureName).then((policyId) => {
                interceptAndWatchRequests(routeMatcherMapForPolicySaveAs)
                    .then(({ waitForRequests }) => {
                        doPolicyRowAction(
                            policyWithNameSelector(importedPolicyFixtureName),
                            'Save as Custom Resource'
                        );
                        cy.get('button:contains("Yes")').click();
                        return waitForRequests();
                    })
                    .then(({ request, response }) => {
                        expect(request.body).to.deep.equal({ ids: [policyId] });
                        expect(response.headers).to.have.property(
                            'content-type',
                            'application/zip'
                        );
                        cy.get(
                            `${selectors.toast.title}:contains("Successfully saved selected policies as Custom Resource")`
                        );
                    });
            });
        });

        it('should allow export of multiple policies via the Bulk actions menu', () => {
            visitPolicies();
            importPolicyFromFixture('policies/good_policy_to_import.json');
            importPolicyFromFixture('policies/good_policy_to_import.json', (contents) => {
                const [firstPolicy, ...rest] = contents.policies;
                return {
                    policies: [
                        {
                            ...firstPolicy,
                            id: window.crypto.randomUUID(),
                            name: `${importedPolicyFixtureName} 2`,
                        },
                        ...rest,
                    ],
                };
            });

            cy.get(selectors.table.bulkActionsDropdownButton).should('be.disabled');
            cy.get(`${policyWithNameSelector(importedPolicyFixtureName)} input[type="checkbox"]`)
                .should('not.be.checked')
                .click();
            cy.get(
                `${policyWithNameSelector(`${importedPolicyFixtureName} 2`)} input[type="checkbox"]`
            )
                .should('not.be.checked')
                .click();
            cy.get(selectors.table.bulkActionsDropdownButton).should('be.enabled').click();

            interceptAndWatchRequests(routeMatcherMapForPolicySaveAs)
                .then(({ waitForRequests }) => {
                    cy.get(
                        `${selectors.table.bulkActionsDropdownItem}:contains("Save as Custom Resource")`
                    ).click();
                    cy.get('button:contains("Yes")').click();
                    return waitForRequests();
                })
                .then(({ request, response }) => {
                    // Request has policy ids.
                    expect(request.body.ids).to.have.length(2);
                    expect(response.headers).to.have.property('content-type', 'application/zip');
                    cy.get(
                        `${selectors.toast.title}:contains("Successfully saved selected policies as Custom Resources")`
                    );
                });
        });

        it('should display toast alert for export service failure', () => {
            visitPolicies();
            importPolicyFromFixture('policies/good_policy_to_import.json');

            interceptAndWatchRequests(routeMatcherMapForPolicySaveAs, {
                [saveAsUrl]: { statusCode: 400 },
            }).then(({ waitForRequests }) => {
                doPolicyRowAction(
                    policyWithNameSelector(importedPolicyFixtureName),
                    'Save as Custom Resource'
                );
                cy.get('button:contains("Yes")').click();
                waitForRequests();
                cy.get(
                    `${selectors.toast.title}:contains("Could not save the selected policies as Custom Resource")`
                );
            });
        });
    });

    describe('policy detail page', () => {
        it('should save policy as custom resource', () => {
            visitPolicies();
            importPolicyFromFixture('policies/good_policy_to_import.json');

            getPolicyIdFromRowWithName(importedPolicyFixtureName).then((policyId) => {
                visitPolicy(policyId);

                interceptAndWatchRequests(routeMatcherMapForPolicySaveAs)
                    .then(({ waitForRequests }) => {
                        doPolicyPageAction('Save as Custom Resource');
                        cy.get('button:contains("Yes")').click();
                        return waitForRequests();
                    })
                    .then(({ request, response }) => {
                        // Request has expected policy id.
                        expect(request.body).to.deep.equal({ ids: [policyId] });
                        expect(response.headers).to.have.property(
                            'content-type',
                            'application/zip'
                        );
                        cy.get(
                            `${selectors.toast.title}:contains("Successfully saved policy as Custom Resource")`
                        );
                    });
            });
        });

        it('should display toast alert for "save as" failure', () => {
            visitPolicies();
            importPolicyFromFixture('policies/good_policy_to_import.json');

            getPolicyIdFromRowWithName(importedPolicyFixtureName).then((policyId) => {
                visitPolicy(policyId);

                interceptAndWatchRequests(routeMatcherMapForPolicySaveAs, {
                    [saveAsUrl]: { statusCode: 400 },
                }).then(({ waitForRequests }) => {
                    doPolicyPageAction('Save as Custom Resource');
                    cy.get('button:contains("Yes")').click();
                    waitForRequests();
                    cy.get(
                        `${selectors.toast.title}:contains("Could not save policy as Custom Resource")`
                    );
                });
            });
        });
    });
});
