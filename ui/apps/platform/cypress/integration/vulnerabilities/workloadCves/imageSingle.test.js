import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';

import {
    applyLocalSeverityFilters,
    typeAndSelectSearchFilterValue,
    selectEntityTab,
    visitWorkloadCveOverview,
    typeAndSelectCustomSearchFilterValue,
} from './WorkloadCves.helpers';
import { selectors } from './WorkloadCves.selectors';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';

describe('Workload CVE Image Single page', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES')) {
            this.skip();
        }
    });

    function visitFirstImage() {
        visitWorkloadCveOverview();

        selectEntityTab('Image');

        // Clear any filters that may be applied to increase the likelihood of finding valid data
        if (hasFeatureFlag('ROX_WORKLOAD_CVES_FIXABILITY_FILTERS')) {
            cy.get(vulnSelectors.clearFiltersButton).click();
        }

        // If unified deferrals are not enabled, there is a good chance none of the visible images will
        // have CVEs, so we apply a wildcard filter to ensure only images with CVEs are visible
        if (!hasFeatureFlag('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL')) {
            typeAndSelectCustomSearchFilterValue('CVE', '.*');
        }

        // Ensure the data in the table has settled
        cy.get(selectors.isUpdatingTable).should('not.exist');

        cy.get('tbody tr td[data-label="Image"] a').first().click();
    }

    it('should contain the correct search filters in the toolbar', () => {
        visitFirstImage();

        // Check that only applicable resource menu items are present in the toolbar
        cy.get(selectors.searchOptionsDropdown).click();
        cy.get(selectors.searchOptionsMenuItem('CVE'));
        cy.get(selectors.searchOptionsMenuItem('Image')).should('not.exist');
        cy.get(selectors.searchOptionsMenuItem('Deployment')).should('not.exist');
        cy.get(selectors.searchOptionsMenuItem('Cluster')).should('not.exist');
        cy.get(selectors.searchOptionsMenuItem('Namespace')).should('not.exist');
        cy.get(selectors.searchOptionsDropdown).click();
    });

    it('should display consistent data between the cards and the table test', () => {
        visitFirstImage();

        // Check that the CVEs by severity totals in the card match the number in the "results found" text
        const cardSelector = selectors.summaryCard('CVEs by severity');
        cy.get(
            [
                `${cardSelector} svg + *:contains("Critical")`,
                `${cardSelector} svg + *:contains("Important")`,
                `${cardSelector} svg + *:contains("Moderate")`,
                `${cardSelector} svg + *:contains("Low")`,
            ].join(',')
        ).then(($severityTotals) => {
            const severityTotal = $severityTotals
                .toArray()
                .reduce((acc, $el) => acc + parseInt($el.innerText.replace(/\D/g, ''), 10), 0);

            const plural = severityTotal === 1 ? '' : 's';
            cy.get(`*:contains(${severityTotal} result${plural} found)`);
        });

        // Check that the CVEs by status totals in the card match the number in the "results found" text
        const fixStatusCardSelector = selectors.summaryCard('CVEs by status');
        cy.get(
            [
                `${fixStatusCardSelector} svg + *:contains("with available fixes")`,
                `${fixStatusCardSelector} svg + *:contains("without fixes")`,
            ].join(',')
        ).then(($statusTotals) => {
            const statusTotal = $statusTotals
                .toArray()
                .reduce((acc, $el) => acc + parseInt($el.innerText.replace(/\D/g, ''), 10), 0);

            const plural = statusTotal === 1 ? '' : 's';
            cy.get(`*:contains(${statusTotal} result${plural} found)`);
        });
    });

    it('should correctly apply a severity filter', () => {
        visitFirstImage();
        // Check that no severities are hidden by default
        cy.get(selectors.summaryCard('CVEs by severity'))
            .find("*:contains('Results hidden')")
            .should('not.exist');

        const severityFilter = 'Critical';

        applyLocalSeverityFilters(severityFilter);

        // Check that summary card severities are hidden correctly
        cy.get(`${selectors.severityIcon('Critical')} + *:contains("Results hidden")`).should(
            'not.exist'
        );
        cy.get(`${selectors.severityIcon('Important')} + *:contains("Results hidden")`);
        cy.get(`${selectors.severityIcon('Moderate')} + *:contains("Results hidden")`);
        cy.get(`${selectors.severityIcon('Low')} + *:contains("Results hidden")`);

        // Check that table rows are filtered
        cy.get(selectors.filteredViewLabel);

        // Ensure the table is not in a loading state
        cy.get(selectors.isUpdatingTable).should('not.exist');

        // Check that every row in the table has the correct severity
        // Query for table rows via jQuery to avoid a Cypress error in the case where there are no rows
        cy.get('table tbody').then(($table) => {
            const $cells = $table.find('tr td[data-label="CVE severity"]');
            // This tests the invariant that if a single severity filter is applied, all rows in the table
            // will have that same severity. This check also holds if the filter removes all rows from the table.
            $cells.each((_, $cell) => {
                const severity = $cell.innerText;
                expect(severity).to.equal(severityFilter);
            });
        });
    });

    // This test should correctly apply a CVE name filter to the CVEs table
    it('should correctly apply CVE name filters', () => {
        visitFirstImage();

        // Get any table row and extract the CVE name from the column with the CVE data label
        cy.get('tbody tr td[data-label="CVE"]')
            .first()
            .then(([$cveNameCell]) => {
                const cveName = $cveNameCell.innerText;
                // Enter the CVE name into the CVE filter
                typeAndSelectSearchFilterValue('CVE', cveName);
                // Check that the header above the table shows only one result
                cy.get(`*:contains("1 result found")`);
                // Check that the only row in the table has the correct CVE name
                cy.get(`tbody tr td[data-label="CVE"]`).should('have.length', 1);
                cy.get(`tbody tr td[data-label="CVE"]:contains("${cveName}")`);
            });
    });
});
