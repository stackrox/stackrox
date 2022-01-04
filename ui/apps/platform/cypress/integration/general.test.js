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
        cy.intercept('GET', api.alerts.countsByCluster).as('alertsByCluster');
    });

    describe('should have correct page titles based on URL', () => {
        const baseTitleText = 'Red Hat Advanced Cluster Security';

        it('for Dashboard', () => {
            cy.intercept('GET', api.dashboard.timeseries).as('dashboardTimeseries');

            cy.visit('/main');
            cy.wait('@dashboardTimeseries');

            cy.title().should('eq', `Dashboard | ${baseTitleText}`);
        });

        it('for Network Graph', () => {
            cy.intercept('GET', api.network.networkGraph).as('networkGraph');

            cy.visit('/main/network');
            cy.wait('@networkGraph');

            cy.title().should('eq', `Network Graph | ${baseTitleText}`);
        });

        it('for Violations', () => {
            cy.intercept('GET', api.alerts.alerts).as('alerts');

            cy.visit('/main/violations');
            cy.wait('@alerts');

            cy.title().should('eq', `Violations | ${baseTitleText}`);
        });

        it('for Violations with side panel open', () => {
            cy.intercept('GET', api.alerts.alertById).as('alertById');

            cy.visit('/main/violations/1234');
            cy.wait('@alertById');

            cy.title().should('eq', `Violations | ${baseTitleText}`);
        });

        it('for Compliance Dashboard', () => {
            const getAggregatedResults = api.graphql(
                api.compliance.graphqlOps.getAggregatedResults
            );
            cy.intercept('POST', getAggregatedResults).as('getAggregatedResults');

            cy.visit('/main/compliance');
            cy.wait('@getAggregatedResults');

            cy.title().should('eq', `Compliance | ${baseTitleText}`);
        });

        it('for Compliance Namespaces', () => {
            const namespaces = api.graphql(api.compliance.graphqlOps.namespaces);
            cy.intercept('POST', namespaces).as('namespaces');

            cy.visit('/main/compliance/namespaces');
            cy.wait('@namespaces');

            cy.title().should('eq', `Compliance - Namespace | ${baseTitleText}`);
        });

        it('for API Docs', () => {
            cy.intercept('GET', api.apiDocs.docs).as('apiDocs');

            cy.visit('/main/apidocs');
            cy.wait('@apiDocs', { timeout: 10000 }); // api docs are sloooooow

            cy.title().should('eq', `API Reference | ${baseTitleText}`);
        });

        it('for User Profile', () => {
            const { mypermissions } = api.roles;
            cy.intercept('GET', mypermissions).as('mypermissions');

            cy.visit('/main/user');
            cy.wait('@mypermissions');

            cy.title().should('eq', `User Profile | ${baseTitleText}`);
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
        cy.get('@firstNavItem').should('have.class', 'pf-m-current');
        cy.get('@firstNavItem').contains('Dashboard');

        // nothing else is selected
        cy.get('@otherNavItems').should('not.have.class', 'pf-m-current');
    });

    // TODO: Fix for ROX-6826
    xit('should have the summary counts in the top header', () => {
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
