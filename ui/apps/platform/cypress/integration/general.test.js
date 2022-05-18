import { url as apidocsUrl } from '../constants/ApiReferencePage';
import { baseURL as complianceUrl } from '../constants/CompliancePage';
import { url as userUrl } from '../constants/UserPage';
import selectors from '../constants/GeneralPage';
import * as api from '../constants/apiEndpoints';
import withAuth from '../helpers/basicAuth';
import { visitMainDashboard, visitMainDashboardFromLeftNav } from '../helpers/main';
import { visitNetworkGraph } from '../helpers/networkGraph';
import { visitViolations, visitViolationsWithUncaughtException } from '../helpers/violations';

//
// Sanity / general checks for UI being up and running
//

describe('General sanity checks', () => {
    withAuth();

    describe('should have correct page titles based on URL', () => {
        // This RegExp allows us to test the format of the page title for specific URLs while
        // staying independent of the product branding in the environment under test.
        const productNameRegExp = '(Red Hat Advanced Cluster Security|StackRox)';

        it('for Dashboard', () => {
            visitMainDashboard();

            cy.title().should('match', new RegExp(`Dashboard | ${productNameRegExp}`));
        });

        it('for Network Graph', () => {
            visitNetworkGraph();

            cy.title().should('match', new RegExp(`Network Graph | ${productNameRegExp}`));
        });

        it('for Violations', () => {
            visitViolations();

            cy.title().should('match', new RegExp(`Violations | ${productNameRegExp}`));
        });

        it('for Compliance Dashboard', () => {
            const getAggregatedResults = api.graphql(
                api.compliance.graphqlOps.getAggregatedResults
            );
            cy.intercept('POST', getAggregatedResults).as('getAggregatedResults');
            cy.visit(complianceUrl);
            cy.wait('@getAggregatedResults');

            cy.title().should('match', new RegExp(`Compliance | ${productNameRegExp}`));
        });

        it('for Compliance Namespaces', () => {
            const namespaces = api.graphql(api.compliance.graphqlOps.namespaces);
            cy.intercept('POST', namespaces).as('namespaces');
            cy.visit(`${complianceUrl}/namespaces`);
            cy.wait('@namespaces');

            cy.title().should('match', new RegExp(`Compliance - Namespace | ${productNameRegExp}`));
        });

        it('for User Profile', () => {
            cy.intercept('GET', api.roles.mypermissions).as('mypermissions');
            cy.intercept('GET', api.auth.authStatus).as('authStatus');
            cy.visit(userUrl);
            cy.wait(['@mypermissions', '@authStatus']);

            cy.title().should('match', new RegExp(`User Profile | ${productNameRegExp}`));
        });

        it('for API Docs', () => {
            // User Profile test often failed when preceded by this test, so move to last place.
            cy.intercept('GET', api.apiDocs.docs).as('apiDocs');
            cy.visit(apidocsUrl);
            cy.wait('@apiDocs', { timeout: 10000 }); // api docs are sloooooow

            cy.title().should('match', new RegExp(`API Reference | ${productNameRegExp}`));
        });
    });

    // TODO: Fix interactive steps for ROX-6826 and merge with the preceding tests to replace visit with assertion about apidocsUrl.
    xit('should go to API docs', () => {
        cy.visit('/');
        cy.get(selectors.navLinks.apidocs).as('apidocs');
        cy.get('@apidocs').click();

        cy.url().should('contain', apidocsUrl);
    });

    it('should allow to navigate to another page after exception happens on a page', () => {
        // Test fails with uncaught exception in local deployment.
        visitViolationsWithUncaughtException();

        cy.get(selectors.errorBoundary).contains(
            "We're sorry — something's gone wrong. The error has been logged."
        );

        visitMainDashboardFromLeftNav();
        cy.get(selectors.errorBoundary).should('not.exist'); // error screen should be gone
    });
});
