import { url as apidocsUrl } from '../constants/ApiReferencePage';
import { url as dashboardUrl } from '../constants/DashboardPage';
import { url as violationsUrl } from '../constants/ViolationsPage';
import selectors from '../constants/GeneralPage';
import * as api from '../constants/apiEndpoints';
import withAuth from '../helpers/basicAuth';

//
// Sanity / general checks for UI being up and running
//

describe('General sanity checks', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.route('GET', api.alerts.countsByCluster).as('alertsByCluster');
    });

    describe('should have correct page titles based on URL', () => {
        it('for Dashboard', () => {
            cy.route('GET', api.dashboard.timeseries).as('dashboardTimeseries');

            cy.visit('/main');
            cy.wait('@dashboardTimeseries');

            cy.title().should('eq', 'Dashboard | StackRox');
        });

        it('for Network Graph', () => {
            cy.route('GET', api.network.networkGraph).as('networkGraph');

            cy.visit('/main/network');
            cy.wait('@networkGraph');

            cy.title().should('eq', 'Network Graph | StackRox');
        });

        it('for Violations', () => {
            cy.route('GET', api.alerts.alerts).as('alerts');

            cy.visit('/main/violations');
            cy.wait('@alerts');

            cy.title().should('eq', 'Violations | StackRox');
        });

        it('for Violations with side panel open', () => {
            cy.route('GET', api.alerts.alertById).as('alertById');

            cy.visit('/main/violations/1234');
            cy.wait('@alertById');

            cy.title().should('eq', 'Violations | StackRox');
        });

        it('for Compliance Dashboard', () => {
            const getAggregatedResults = api.graphql(
                api.compliance.graphqlOps.getAggregatedResults
            );
            cy.route('POST', getAggregatedResults).as('getAggregatedResults');

            cy.visit('/main/compliance');
            cy.wait('@getAggregatedResults');

            cy.title().should('eq', 'Compliance | StackRox');
        });

        it('for Compliance Namespaces', () => {
            const namespaces = api.graphql(api.compliance.graphqlOps.namespaces);
            cy.route('POST', namespaces).as('namespaces');

            cy.visit('/main/compliance/namespaces');
            cy.wait('@namespaces');

            cy.title().should('eq', 'Compliance - Namespace | StackRox');
        });

        it('for API Docs', () => {
            cy.route('GET', api.apiDocs.docs).as('apiDocs');

            cy.visit('/main/apidocs');
            cy.wait('@apiDocs', { timeout: 10000 }); // api docs are sloooooow

            cy.title().should('eq', 'API Reference | StackRox');
        });

        it('for User Page', () => {
            const summaryCounts = api.graphql(api.general.graphqlOps.summaryCounts);
            cy.route('POST', summaryCounts).as('summaryCounts');

            cy.visit('/main/user');
            cy.wait('@summaryCounts');

            cy.title().should('eq', 'User Page | StackRox');
        });

        it('for License Page', () => {
            cy.route('GET', api.licenses.list).as('licenses');

            cy.visit('/main/license');
            cy.wait('@licenses');

            cy.title().should('eq', 'License | StackRox');
        });
    });

    it('should render navbar with Dashboard selected', () => {
        cy.visit('/');
        cy.wait('@alertsByCluster');

        cy.get(selectors.navLinks.first).as('firstNavItem');
        cy.get(selectors.navLinks.others).as('otherNavItems');

        // redirect should happen
        cy.url().should('contain', dashboardUrl);

        // Dashboard is selected
        cy.get('@firstNavItem').should('have.class', 'bg-primary-700');
        cy.get('@firstNavItem').contains('Dashboard');

        // nothing else is selected
        cy.get('@otherNavItems').should('not.have.class', 'bg-primary-700');

        cy.get(selectors.navLinks.list).as('topNavItems');
        cy.get('@topNavItems').should(($lis) => {
            expect($lis).to.have.length(6);
            expect($lis.eq(0)).to.contain('Cluster');
            expect($lis.eq(1)).to.contain('Node');
            expect($lis.eq(2)).to.contain('Violation');
            expect($lis.eq(3)).to.contain('Deployment');
            expect($lis.eq(4)).to.contain('Image');
            expect($lis.eq(5)).to.contain('Secret');
        });
    });

    it('should go to API docs', () => {
        cy.visit('/');
        cy.get(selectors.navLinks.apidocs).as('apidocs');
        cy.get('@apidocs').click();

        cy.url().should('contain', apidocsUrl);
    });

    it('should allow to navigate to another page after exception happens on a page', () => {
        cy.server();
        cy.route('GET', api.alerts.alerts, { alerts: [{ id: 'broken one' }] }).as('alerts');

        cy.visit(violationsUrl);
        cy.wait('@alerts');

        cy.get(selectors.errorBoundary).contains(
            "We're sorry — something's gone wrong. The error has been logged."
        );

        cy.get(selectors.navLinks.first).click();
        cy.get(selectors.errorBoundary).should('not.exist'); // error screen should be gone
    });
});
