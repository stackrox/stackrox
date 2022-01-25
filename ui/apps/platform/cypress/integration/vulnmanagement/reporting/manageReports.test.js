import * as api from '../../../constants/apiEndpoints';
import { url, selectors } from '../../../constants/VulnManagementPage';
import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';

describe('Vulnmanagement reports', () => {
    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_VULN_REPORTING')) {
            this.skip();
        }
    });

    withAuth();

    describe('creating a report', () => {
        beforeEach(() => {
            cy.intercept('GET', api.report.configurations, { reportConfigs: [] }).as(
                'getReportConfigurations'
            );
        });

        it('should navigate to the Create Report view by button or directly', () => {
            cy.visit('/main/dashboard');
            cy.get(selectors.vulnManagementExpandableNavLink).click({ force: true });
            cy.get(selectors.vulnManagementExpandedReportingNavLink).click({ force: true });
            cy.url().should('contain', url.reporting.list);

            // navigate by button
            cy.wait('@getReportConfigurations');
            cy.get(selectors.reportSection.createReportLink).click();
            cy.location('pathname').should('eq', `${url.reporting.list}`);
            cy.location('search').should('eq', '?action=create');

            // check the breadcrumbs
            cy.get(selectors.reportSection.breadcrumbItems)
                .last()
                .contains('Create a vulnerability report');
            // first breadcrumb should be link back to reports table
            cy.get(selectors.reportSection.breadcrumbItems).first().click();
            cy.location('pathname').should('eq', `${url.reporting.list}`);

            // navigate directly by URL
            cy.visit('/main/dashboard'); // leave Create Report page
            cy.visit(`${url.reporting.list}?action=create`);
            cy.get('h1:contains("Create a vulnerability report")');
        });
    });
});
