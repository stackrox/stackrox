import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';

import {
    applyLocalSeverityFilters,
    selectEntityTab,
    visitWorkloadCveOverview,
} from './WorkloadCves.helpers';
import { selectors } from './WorkloadCves.selectors';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';

describe('Workload CVE Deployment Single page', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES')) {
            this.skip();
        }
    });

    function visitFirstDeployment() {
        visitWorkloadCveOverview();

        // Clear any filters that may be applied to increase the likelihood of finding a deployment
        if (hasFeatureFlag('ROX_WORKLOAD_CVES_FIXABILITY_FILTERS')) {
            cy.get(vulnSelectors.clearFiltersButton).click();
        }

        selectEntityTab('Deployment');
        cy.get('tbody tr td[data-label="Deployment"] a').first().click();
    }

    it('should contain the correct search filters in the toolbar', () => {
        visitFirstDeployment();

        // Check that only applicable resource menu items are present in the toolbar
        cy.get(selectors.searchOptionsDropdown).click();
        cy.get(selectors.searchOptionsMenuItem('CVE'));
        cy.get(selectors.searchOptionsMenuItem('Image'));
        cy.get(selectors.searchOptionsMenuItem('Component'));
        cy.get(selectors.searchOptionsMenuItem('Component source'));
        cy.get(selectors.searchOptionsMenuItem('Deployment')).should('not.exist');
        cy.get(selectors.searchOptionsMenuItem('Cluster')).should('not.exist');
        cy.get(selectors.searchOptionsMenuItem('Namespace')).should('not.exist');
        cy.get(selectors.searchOptionsDropdown).click();
    });

    it('should navigate between vulnerabilities and resources tabs', () => {
        visitFirstDeployment();

        // By default, the vulnerabilities tab should be selected
        cy.get(selectors.vulnerabilitiesTab).should('have.attr', 'aria-selected', 'true');
        cy.get(selectors.resourcesTab).should('have.attr', 'aria-selected', 'false');

        // Check elements on the Vulnerabilities tab
        cy.get(selectors.summaryCard('CVEs by severity'));
        cy.get(selectors.summaryCard('CVEs by status'));
        // Check table exists with the correct headers
        cy.get('table thead tr th').contains('CVE');
        cy.get('table thead tr th').contains('CVE severity');
        cy.get('table thead tr th').contains('CVE status');
        cy.get('table thead tr th').contains('Affected components');
        cy.get('table thead tr th').contains('First discovered');

        // Visit the resources tab
        cy.get(selectors.resourcesTab).click();

        cy.get(selectors.vulnerabilitiesTab).should('have.attr', 'aria-selected', 'false');
        cy.get(selectors.resourcesTab).should('have.attr', 'aria-selected', 'true');

        // Check items on the Resources tab
        // Check table exists with the correct headers
        cy.get('table thead tr th').contains('Name');
        cy.get('table thead tr th').contains('Image status');
        cy.get('table thead tr th').contains('Image OS');
        cy.get('table thead tr th').contains('Created');
    });

    it('should reset pagination when navigating between resources and vulnerabilities tabs', () => {
        visitFirstDeployment();

        // Check that pagination is reset across tabs
        // 1. Manually reload page with a page query param to ensure that the page is not 1
        // 2. Visit the resources tab
        // 3. Ensure that the page query param is removed from the URL
        // 4. Repeat steps 1-3 for the vulnerabilities tab
        cy.location('href').then((href) => {
            cy.visit(`${href}?page=2`);
            cy.get(selectors.resourcesTab).click();
            cy.location('search').should('not.contain', 'page=');
        });

        cy.location('href').then((href) => {
            cy.visit(`${href}?page=2`);
            cy.get(selectors.vulnerabilitiesTab).click();
            cy.location('search').should('not.contain', 'page=');
        });
    });

    it('should handle applied filter behavior', () => {
        visitFirstDeployment();

        // Check that no severities are hidden by default
        cy.get(selectors.summaryCard('CVEs by severity'))
            .find("*:contains('Results hidden')")
            .should('not.exist');

        applyLocalSeverityFilters('Critical');

        // Check that summary card severities are hidden correctly
        cy.get(`${selectors.severityIcon('Critical')} + *:contains("Results hidden")`).should(
            'not.exist'
        );
        cy.get(`${selectors.severityIcon('Important')} + *:contains("Results hidden")`);
        cy.get(`${selectors.severityIcon('Moderate')} + *:contains("Results hidden")`);
        cy.get(`${selectors.severityIcon('Low')} + *:contains("Results hidden")`);

        // Check that table rows are filtered
        cy.get(selectors.filteredViewLabel);
    });
});
