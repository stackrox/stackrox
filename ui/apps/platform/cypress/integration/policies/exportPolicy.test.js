import * as api from '../../constants/apiEndpoints';
import { selectors } from '../../constants/PoliciesPage';
import withAuth from '../../helpers/basicAuth';
import {
    doPolicyPageAction,
    doPolicyRowAction,
    visitPolicies,
    visitPolicy,
} from '../../helpers/policies';

describe('Export policy', () => {
    withAuth();

    describe('policies table', () => {
        it('should export policy', () => {
            visitPolicies();

            const trSelector = 'tbody:nth-child(2) tr:nth-child(1)';
            cy.get(`${trSelector} ${selectors.table.policyLink}`).then(($a) => {
                const segments = $a.attr('href').split('/');
                const policyId = segments[segments.length - 1];

                cy.intercept('POST', api.policies.export).as('exportPolicy');

                doPolicyRowAction(trSelector, 'Export policy to JSON');

                cy.wait('@exportPolicy').then(({ request, response }) => {
                    // Request has expected policy id.
                    expect(request.body).to.deep.equal({
                        policyIds: [policyId],
                    });

                    // Response has expected policy id.
                    expect(response.body.policies).to.have.length(1);
                    expect(response.body.policies[0]).to.include({
                        id: policyId,
                    });
                });
                cy.get(`${selectors.toast.title}:contains("Successfully exported policy")`);
            });
        });

        it('should display toast alert for export request failure', () => {
            visitPolicies();

            const trSelector = 'tbody:nth-child(2) tr:nth-child(1)';
            const message = 'Some policies could not be retrieved.';
            cy.intercept('POST', api.policies.export, {
                statusCode: 400,
                body: {
                    message, // emulate request failure
                },
            }).as('exportPolicy');

            doPolicyRowAction(trSelector, 'Export policy to JSON');

            cy.wait('@exportPolicy');
            cy.get(`${selectors.toast.title}:contains("Could not export the policy")`);
            cy.get(`${selectors.toast.description}:contains("${message}")`);
        });

        it('should display toast alert for export service failure', () => {
            visitPolicies();

            const trSelector = 'tbody:nth-child(2) tr:nth-child(1)';
            const message = 'Problem exporting policy data';
            cy.intercept('POST', api.policies.export, {
                statusCode: 400,
                body: {
                    message, // emulate error thrown by service call after request success
                },
            }).as('exportPolicy');

            doPolicyRowAction(trSelector, 'Export policy to JSON');

            cy.wait('@exportPolicy');
            cy.get(`${selectors.toast.title}:contains("Could not export the policy")`);
            cy.get(`${selectors.toast.description}:contains("${message}")`);
        });
    });

    describe('policy page', () => {
        it('should export policy', () => {
            visitPolicies();

            const trSelector = 'tbody tr:nth-child(1)';
            cy.get(`${trSelector} ${selectors.table.policyLink}`).then(($a) => {
                const segments = $a.attr('href').split('/');
                const policyId = segments[segments.length - 1];

                visitPolicy(policyId);

                cy.intercept('POST', api.policies.export).as('exportPolicy');

                doPolicyPageAction('Export policy to JSON');

                cy.wait('@exportPolicy').then(({ request, response }) => {
                    // Request has expected policy id.
                    expect(request.body).to.deep.equal({
                        policyIds: [policyId],
                    });

                    // Response has expected policy id.
                    expect(response.body.policies).to.have.length(1);
                    expect(response.body.policies[0]).to.include({
                        id: policyId,
                    });
                });
                cy.get(`${selectors.toast.title}:contains("Successfully exported policy")`);
            });
        });

        it('should display toast alert for export request failure', () => {
            visitPolicies();

            const trSelector = 'tbody tr:nth-child(1)';
            cy.get(`${trSelector} ${selectors.table.policyLink}`).then(($a) => {
                const segments = $a.attr('href').split('/');
                const policyId = segments[segments.length - 1];

                visitPolicy(policyId);

                const message = 'Some policies could not be retrieved.';
                cy.intercept('POST', api.policies.export, {
                    statusCode: 400,
                    body: {
                        message, // emulate request failure
                    },
                }).as('exportPolicy');

                doPolicyPageAction('Export policy to JSON');

                cy.wait('@exportPolicy');
                cy.get(`${selectors.toast.title}:contains("Could not export the policy")`);
                cy.get(`${selectors.toast.description}:contains("${message}")`);
            });
        });

        it('should display toast alert for export service failure', () => {
            visitPolicies();

            const trSelector = 'tbody tr:nth-child(1)';
            cy.get(`${trSelector} ${selectors.table.policyLink}`).then(($a) => {
                const segments = $a.attr('href').split('/');
                const policyId = segments[segments.length - 1];

                visitPolicy(policyId);

                const message = 'Problem exporting policy data';
                cy.intercept('POST', api.policies.export, {
                    statusCode: 400,
                    body: {
                        message, // emulate error thrown by service call after request success
                    },
                }).as('exportPolicy');

                doPolicyPageAction('Export policy to JSON');

                cy.wait('@exportPolicy');
                cy.get(`${selectors.toast.title}:contains("Could not export the policy")`);
                cy.get(`${selectors.toast.description}:contains("${message}")`);
            });
        });
    });
});
