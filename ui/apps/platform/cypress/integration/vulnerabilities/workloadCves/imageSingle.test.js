import upperFirst from 'lodash/upperFirst';

import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';

import {
    applyLocalSeverityFilters,
    typeAndSelectResourceFilterValue,
    selectEntityTab,
    visitWorkloadCveOverview,
} from './WorkloadCves.helpers';
import { selectors } from './WorkloadCves.selectors';

describe('Workload CVE Image Single page', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES')) {
            this.skip();
        }
    });

    it('should correctly handle Image single page specific behavior', () => {
        visitWorkloadCveOverview();

        selectEntityTab('Image');
        cy.get('tbody tr td[data-label="Image"] a').first().click();

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
        visitWorkloadCveOverview();

        selectEntityTab('Image');
        // Find any image with at least one CVE

        cy.get(
            `tbody tr:has(td[data-label="CVEs by severity"] ${selectors.nonZeroCveSeverityCounts}) td[data-label="Image"] a`
        )
            .first()
            .click();

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

    it('should correctly apply severity filters', () => {
        visitWorkloadCveOverview();

        selectEntityTab('Image');

        // Find any image with at least one CVE
        cy.get(
            `tbody tr:has(td[data-label="CVEs by severity"] ${selectors.nonZeroCveSeverityCounts})`
        )
            .first()
            .then(([$rowWithCves]) => {
                // Get a nonzero count and severity from the table for the selected image
                cy.wrap($rowWithCves)
                    .get(selectors.nonZeroCveSeverityCounts)
                    .then(([$nonZeroSeverityLabel]) => {
                        const ariaLabel = $nonZeroSeverityLabel.getAttribute('aria-label');
                        const [, countRaw, severityRaw] = ariaLabel.match(/(\d+) (\w+)/);
                        const count = parseInt(countRaw, 10);
                        const severity = upperFirst(severityRaw);

                        const bySeverityCard = selectors.summaryCard('CVEs by severity');
                        const byStatusCard = selectors.summaryCard('CVEs by status');

                        // Click the link in the table to visit the image page
                        cy.wrap($rowWithCves).find('td[data-label="Image"] a').click();

                        // Check the severity and count in the summary card against the data from the original table
                        cy.get(`${bySeverityCard} ${selectors.iconText(`${count} ${severity}`)}`);

                        // Apply a severity filter that matches the chosen severity
                        applyLocalSeverityFilters(severity);

                        // Check that all of the other severities in the card read "Results hidden"
                        cy.get(`${bySeverityCard} ${selectors.iconText(severity)}`);
                        cy.get(`${bySeverityCard} ${selectors.iconText('Results hidden')}`).should(
                            'have.length',
                            3
                        );

                        // Check that the filtered view label is present
                        cy.get(selectors.filteredViewLabel);

                        // Check that the row count in the header above the table matches the CVE count for the image
                        cy.get(`*`).contains(new RegExp(`${count} results? found`));

                        // Check that the count and severity in the summary card still match after the filter is applied
                        cy.get(`${bySeverityCard} ${selectors.iconText(`${count} ${severity}`)}`);

                        // Check that the total number of fixable + not fixable matches the total number of CVEs
                        cy.get(
                            [
                                `${byStatusCard} ${selectors.iconText('with available fixes')}`,
                                `${byStatusCard} ${selectors.iconText('without fixes')}`,
                            ].join(',')
                        ).then(([$fixable, $notFixable]) => {
                            const fixableCount = parseInt(
                                $fixable.innerText.replace(/\D/g, ''),
                                10
                            );
                            const notFixableCount = parseInt(
                                $notFixable.innerText.replace(/\D/g, ''),
                                10
                            );
                            expect(fixableCount + notFixableCount).to.equal(count);
                        });

                        // Check that every row in the table has the correct severity
                        cy.get(`table tbody tr td[data-label="CVE severity"]`).each(($severity) => {
                            expect($severity.text()).to.equal(severity);
                        });
                    });
            });
    });

    // This test should correctly apply a CVE name filter to the CVEs table
    it('should correctly apply CVE name filters', () => {
        // Visit the workload CVE overview page
        visitWorkloadCveOverview();

        selectEntityTab('Image');

        // Select any image that has CVEs in the table
        // and click the link in the row to visit the image page
        cy.get(
            `tbody tr:has(td[data-label="CVEs by severity"] ${selectors.nonZeroCveSeverityCounts})`
        )
            .first()
            .find('td[data-label="Image"] a')
            .click();

        // Get any table row and extract the CVE name from the column with the CVE data label
        cy.get('tbody tr td[data-label="CVE"]')
            .first()
            .then(([$cveNameCell]) => {
                const cveName = $cveNameCell.innerText;
                // Enter the CVE name into the CVE filter
                typeAndSelectResourceFilterValue('CVE', cveName);
                // Check that the header above the table shows only one result
                cy.get(`*:contains("1 result found")`);
                // Check that the only row in the table has the correct CVE name
                cy.get(`tbody tr td[data-label="CVE"]`).should('have.length', 1);
                cy.get(`tbody tr td[data-label="CVE"]:contains("${cveName}")`);
            });
    });
});
