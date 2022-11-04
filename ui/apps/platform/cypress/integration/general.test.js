import { url as userUrl } from '../constants/UserPage';
import withAuth from '../helpers/basicAuth';
import { visitComplianceDashboard, visitComplianceEntities } from '../helpers/compliance';
import { visitMainDashboard } from '../helpers/main';
import { visitNetworkGraph } from '../helpers/networkGraph';
import { visitViolations } from '../helpers/violations';
import { visit } from '../helpers/visit';

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
    });
});
