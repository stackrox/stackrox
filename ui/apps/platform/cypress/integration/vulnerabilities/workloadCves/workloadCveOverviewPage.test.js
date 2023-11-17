import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import {
    applyLocalSeverityFilters,
    extractNonZeroSeverityFromCount,
    selectEntityTab,
    visitWorkloadCveOverview,
} from './WorkloadCves.helpers';

import { selectors } from './WorkloadCves.selectors';

describe('Workload CVE overview page tests', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES')) {
            this.skip();
        }
    });

    it('should satisfy initial page load defaults', () => {
        visitWorkloadCveOverview();

        // TODO Test that the default tab is set to "Observed"

        // Check that the CVE entity toggle is selected and Image/Deployment are disabled
        cy.get(selectors.entityTypeToggleItem('CVE')).should('have.attr', 'aria-pressed', 'true');
        cy.get(selectors.entityTypeToggleItem('Image')).should(
            'not.have.attr',
            'aria-pressed',
            'true'
        );
        cy.get(selectors.entityTypeToggleItem('Deployment')).should(
            'not.have.attr',
            'aria-pressed',
            'true'
        );
    });

    it('should correctly handle applied filters across entity tabs', function () {
        if (!hasFeatureFlag('ROX_WORKLOAD_CVES_FIXABILITY_FILTERS')) {
            this.skip();
        }
        visitWorkloadCveOverview();

        // We want to manually test filter application, so clear the default filters
        cy.get(selectors.clearFiltersButton).click();
        cy.get(selectors.hiddenSeverityCount('Critical')).should('not.exist');

        // Get the first CVE row from the table with a non-zero severity count for -any- severity
        cy.get(selectors.nonZeroImageSeverityCounts)
            .first()
            .then(($severityCount) => {
                const [nonZeroSeverity, unusedSeverities] = extractNonZeroSeverityFromCount(
                    $severityCount.attr('aria-label')
                );

                expect(unusedSeverities).to.have.lengthOf(3);

                // Apply the severity filter for the first non-zero CVE severity count
                // @ts-ignore
                applyLocalSeverityFilters(nonZeroSeverity);

                // Check that the filter chip for the severity exists
                cy.get(selectors.filterChipGroupItem('Severity', nonZeroSeverity));

                // Check that the table is no longer updating the data to reflect the new filter
                cy.get(selectors.isUpdatingTable).should('not.exist');

                // Check that all table rows have a non-zero severity count for the selected severity
                // Check that all table rows have other severity counts hidden for all other severities
                const hiddenSeveritySelectors = [
                    selectors.hiddenSeverityCount(unusedSeverities[0]),
                    selectors.hiddenSeverityCount(unusedSeverities[1]),
                    selectors.hiddenSeverityCount(unusedSeverities[2]),
                ];
                const imageSeverityCountSelector = [
                    selectors.nonZeroImageSeverityCount(nonZeroSeverity),
                    ...hiddenSeveritySelectors,
                ].join(',');

                const cveSeverityCountSelector = [
                    selectors.nonZeroCveSeverityCount(nonZeroSeverity),
                    ...hiddenSeveritySelectors,
                ].join(',');

                // Check all rows for the CVE table
                // TODO David can you rewrite with safe chain?
                /* eslint-disable cypress/unsafe-to-chain-command */
                cy.get('table tbody tr:nth-of-type(1)')
                    .each(($row) => cy.wrap($row).find(imageSeverityCountSelector))
                    // Check all rows for the Image table
                    .then(() => {
                        selectEntityTab('Image');
                        return cy.get('table tbody tr:nth-of-type(1)');
                    })
                    .each(($row) => cy.wrap($row).find(cveSeverityCountSelector))
                    // Check all rows for the Deployment table
                    .then(() => {
                        selectEntityTab('Deployment');
                        return cy.get('table tbody tr:nth-of-type(1)');
                    })
                    .each(($row) => cy.wrap($row).find(cveSeverityCountSelector));
                /* eslint-disable cypress/unsafe-to-chain-command */
            });
    });
});
