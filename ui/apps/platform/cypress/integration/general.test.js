import { url as apidocsUrl } from '../constants/ApiReferencePage';
import { url as userUrl } from '../constants/UserPage';
import selectors from '../constants/GeneralPage';
import * as api from '../constants/apiEndpoints';
import withAuth from '../helpers/basicAuth';
import { visitComplianceDashboard, visitComplianceEntities } from '../helpers/compliance';
import { visitMainDashboard, visitMainDashboardFromLeftNav } from '../helpers/main';
import { visitNetworkGraph } from '../helpers/networkGraph';
import { visit } from '../helpers/visit';
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
            visitComplianceDashboard();

            cy.title().should('match', new RegExp(`Compliance | ${productNameRegExp}`));
        });

        it('for Compliance Namespaces', () => {
            visitComplianceEntities('namespaces');

            cy.title().should('match', new RegExp(`Compliance - Namespace | ${productNameRegExp}`));
        });

        it('for User Profile', () => {
            visit(userUrl);

            cy.title().should('match', new RegExp(`User Profile | ${productNameRegExp}`));
        });

        it('for API Docs', () => {
            // User Profile test often failed when preceded by this test, so move to last place.
            cy.intercept('GET', api.apiDocs.docs).as('apiDocs');
            visit(apidocsUrl);
            cy.wait('@apiDocs', { responseTimeout: 10000 }); // api docs are sloooooow

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
