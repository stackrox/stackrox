import { url as apidocsUrl } from '../constants/ApiReferencePage';
import { baseURL as complianceUrl } from '../constants/CompliancePage';
import { url as dashboardUrl, selectors as dashboardSelectors } from '../constants/DashboardPage';
import { url as networkUrl } from '../constants/NetworkPage';
import { url as userUrl } from '../constants/UserPage';
import { url as violationsUrl } from '../constants/ViolationsPage';
import selectors from '../constants/GeneralPage';
import * as api from '../constants/apiEndpoints';
import withAuth from '../helpers/basicAuth';

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
            cy.intercept('POST', api.dashboard.summaryCounts).as('summaryCounts');
            cy.visit(dashboardUrl);
            cy.wait('@summaryCounts');

            cy.title().should('match', new RegExp(`Dashboard | ${productNameRegExp}`));
        });

        it('for Network Graph', () => {
            cy.intercept('GET', api.network.networkGraph).as('networkGraph');
            cy.intercept('GET', api.network.networkPoliciesGraph).as('networkPolicies');
            cy.visit(networkUrl);
            cy.wait(['@networkGraph', '@networkPolicies']);

            cy.title().should('match', new RegExp(`Network Graph | ${productNameRegExp}`));
        });

        it('for Violations', () => {
            cy.intercept('GET', api.alerts.alerts).as('alerts');
            cy.intercept('GET', api.alerts.alertscount).as('alertsCount');
            cy.visit(violationsUrl);
            cy.wait(['@alerts', '@alertsCount']);

            cy.title().should('match', new RegExp(`Violations | ${productNameRegExp}`));
        });

        it('for Violations with side panel open', () => {
            cy.intercept('GET', api.alerts.alertById).as('alertById');
            cy.visit('/main/violations/1234');
            cy.wait('@alertById'); // 404

            cy.title().should('match', new RegExp(`Violations | ${productNameRegExp}`)); // Violation not found.
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

    it('should render navbar with Dashboard selected', () => {
        cy.intercept('POST', api.dashboard.summaryCounts).as('summaryCounts');
        cy.visit('/');
        cy.wait('@summaryCounts');

        // redirect should happen
        cy.location('pathname').should('eq', dashboardUrl);

        // Dashboard is selected
        cy.get(selectors.navLinks.first).should('have.class', 'pf-m-current');
        cy.get(selectors.navLinks.first).contains('Dashboard');

        // nothing else is selected
        cy.get(selectors.navLinks.others).should('not.have.class', 'pf-m-current');
    });

    it('should have the summary counts in the top header', () => {
        cy.intercept('POST', api.dashboard.summaryCounts).as('summaryCounts');
        cy.visit(dashboardUrl);
        cy.wait('@summaryCounts');

        const { summaryCount: summaryCountSelector } = dashboardSelectors;
        cy.get(`${summaryCountSelector}:nth-child(1):contains("Cluster")`);
        cy.get(`${summaryCountSelector}:nth-child(2):contains("Node")`);
        cy.get(`${summaryCountSelector}:nth-child(3):contains("Violation")`);
        cy.get(`${summaryCountSelector}:nth-child(4):contains("Deployment")`);
        cy.get(`${summaryCountSelector}:nth-child(5):contains("Image")`);
        cy.get(`${summaryCountSelector}:nth-child(6):contains("Secret")`);
    });

    // TODO: Fix for ROX-6826
    xit('should go to API docs', () => {
        cy.visit('/');
        cy.get(selectors.navLinks.apidocs).as('apidocs');
        cy.get('@apidocs').click();

        cy.url().should('contain', apidocsUrl);
    });

    it('should allow to navigate to another page after exception happens on a page', () => {
        cy.intercept('GET', api.alerts.alerts, {
            body: { alerts: [{ id: 'broken one' }] },
        }).as('alerts');

        cy.visit(violationsUrl);
        cy.wait('@alerts');

        cy.get(selectors.errorBoundary).contains(
            "We're sorry — something's gone wrong. The error has been logged."
        );

        cy.get(selectors.navLinks.first).click();
        cy.get(selectors.errorBoundary).should('not.exist'); // error screen should be gone
    });
});
