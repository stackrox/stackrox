import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import { graphql } from '../../../constants/apiEndpoints';
import {
    applyLocalSeverityFilters,
    selectEntityTab,
    visitWorkloadCveOverview,
    typeAndSelectCustomSearchFilterValue,
    changeObservedCveViewingMode,
    interactAndWaitForCveList,
    interactAndWaitForImageList,
    interactAndWaitForDeploymentList,
    typeAndEnterCustomSearchFilterValue,
} from './WorkloadCves.helpers';
import { selectors } from './WorkloadCves.selectors';

describe('Workload CVE overview page tests', () => {
    const isAdvancedFiltersEnabled = hasFeatureFlag('ROX_VULN_MGMT_ADVANCED_FILTERS');

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

    it('should correctly handle applied filters across entity tabs', () => {
        visitWorkloadCveOverview();

        const entityOpnameMap = {
            CVE: 'getImageCVEList',
            Image: 'getImageList',
            Deployment: 'getDeploymentList',
        };

        const { CVE, Image, Deployment } = entityOpnameMap;

        // Intercept and mock responses as empty, since we don't care about the response
        cy.intercept({ method: 'POST', url: graphql(CVE) }, { data: {} }).as(CVE);
        cy.intercept({ method: 'POST', url: graphql(Image) }, { data: {} }).as(Image);
        cy.intercept({ method: 'POST', url: graphql(Deployment) }, { data: {} }).as(Deployment);

        applyLocalSeverityFilters('Critical');

        // Test that the correct filters are applied for each entity tab, and that the correct
        // search filter is sent in the request for each tab
        Object.entries(entityOpnameMap).forEach(([entity /*, opname */]) => {
            // @ts-ignore
            selectEntityTab(entity);

            // Ensure that only the correct filter chip is present
            const filterChipGroupName = isAdvancedFiltersEnabled ? 'CVE severity' : 'Severity';
            cy.get(selectors.filterChipGroupItem(filterChipGroupName, 'Critical'));
            cy.get(selectors.filterChipGroupItems).should('have.lengthOf', 1);

            // TODO - See if there is a clean way to re-enable this to handle both cases where the
            // feature flag is not enabled and not enabled
            /*
            // Ensure the correct search filter is present in the request
            cy.wait(`@${opname}`).should((xhr) => {
                expect(xhr.request.body.variables.query).to.contain(
                    'SEVERITY:CRITICAL_VULNERABILITY_SEVERITY'
                );
            });
            */
        });
    });

    describe('Images without CVEs view tests', () => {
        beforeEach(function () {
            if (
                !hasFeatureFlag('ROX_WORKLOAD_CVES_FIXABILITY_FILTERS') ||
                !hasFeatureFlag('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL')
            ) {
                this.skip();
            }
        });

        it('should remove cve-related UI elements when viewing the "without cves" view', () => {
            visitWorkloadCveOverview();

            // TODO
            // These cannot be relied on in CI until the table is refactored to use
            // the new table component that always renders the header
            // const riskPriorityHeader = 'th:contains("Risk priority")';
            // const cvesBySeverityHeader = 'th:contains("CVEs by severity")';
            const prioritizeByNamespaceButton = 'button:contains("Prioritize by namespace view")';
            const defaultFiltersButton = 'button:contains("Default filters")';

            function assertCveElementsArePresent() {
                // TODO
                // These cannot be relied on in CI until the table is refactored to use
                // the new table component that always renders the header
                // cy.get(riskPriorityHeader);
                // cy.get(cvesBySeverityHeader);
                cy.get(prioritizeByNamespaceButton);
                cy.get(defaultFiltersButton);
                cy.get(selectors.severityDropdown);
                cy.get(selectors.fixabilityDropdown);
            }

            function assertCveElementsAreNotPresent() {
                // TODO
                // These cannot be relied on in CI until the table is refactored to use
                // the new table component that always renders the header
                // cy.get(riskPriorityHeader).should('not.exist');
                // cy.get(cvesBySeverityHeader).should('not.exist');
                cy.get(prioritizeByNamespaceButton).should('not.exist');
                cy.get(defaultFiltersButton).should('not.exist');
                cy.get(selectors.severityDropdown).should('not.exist');
                cy.get(selectors.fixabilityDropdown).should('not.exist');
            }

            // Visit the Images tab
            interactAndWaitForImageList(() => {
                selectEntityTab('Image');
            });

            assertCveElementsArePresent();

            // Visit the Images tab
            interactAndWaitForDeploymentList(() => {
                selectEntityTab('Deployment');
            });

            assertCveElementsArePresent();

            // Switch to the "without cves" view, we should stay on the deployments tab
            interactAndWaitForDeploymentList(() => {
                changeObservedCveViewingMode('Images without vulnerabilities');
            });

            assertCveElementsAreNotPresent();

            // Visit the Images tab
            interactAndWaitForImageList(() => {
                selectEntityTab('Image');
            });

            assertCveElementsAreNotPresent();
        });

        it('should apply the correct filters when switching between "with cves" and "without cves" views', () => {
            const severityChip = isAdvancedFiltersEnabled ? 'CVE severity' : 'Severity';
            const cveStatusChip = 'CVE status';
            const imageNameChip = isAdvancedFiltersEnabled ? 'Image name' : 'Image';

            // Since we want to test the behavior of the default filters with the two cve views, we
            // do not clear them by default in this case
            visitWorkloadCveOverview({ clearFiltersOnVisit: false });

            interactAndWaitForCveList(() => {
                // Add a local filter
                if (isAdvancedFiltersEnabled) {
                    typeAndEnterCustomSearchFilterValue('Image', 'Name', 'quay.io/bogus');
                } else {
                    typeAndSelectCustomSearchFilterValue('Image', 'quay.io/bogus');
                }

                // Check that default filters and the local filter are present
                cy.get(selectors.filterChipGroupItem(severityChip, 'Critical'));
                cy.get(selectors.filterChipGroupItem(severityChip, 'Important'));
                cy.get(selectors.filterChipGroupItem(cveStatusChip, 'Fixable'));
                cy.get(selectors.filterChipGroupItem(imageNameChip, 'quay.io/bogus'));
            }).should((xhr) => {
                // Ensure the default "with cves" view passes a "Vulnerability State" filter automatically
                // Ensure default and local filters are passed as well
                const requestQuery = xhr.request.body.variables.query.toLowerCase();
                expect(requestQuery).to.contain('vulnerability state');
                expect(requestQuery).to.contain(
                    'severity:critical_vulnerability_severity,important_vulnerability_severity'
                );
                expect(requestQuery).to.contain('fixable:true');
                expect(requestQuery).to.contain('image:r/quay.io/bogus');
                // This view should not filter to
                expect(requestQuery).not.to.contain('image cve count');
            });

            interactAndWaitForImageList(() => {
                // Change to the "without cves" view, note that since we are currently on the
                // CVE tab, we should automatically switch to the Image tab
                changeObservedCveViewingMode('Images without vulnerabilities');

                // Filters should be cleared
                cy.get(selectors.filterChipGroup).should('not.exist');
            }).should((xhr) => {
                // On switching views, all filters, including the defaults should be cleared
                const requestQuery = xhr.request.body.variables.query.toLowerCase();
                // The request should complete with only a filter for images without cves
                expect(requestQuery).to.equal('image cve count:0');
            });

            interactAndWaitForImageList(() => {
                // Apply a filter in the "without cves" view
                if (isAdvancedFiltersEnabled) {
                    typeAndEnterCustomSearchFilterValue('Image', 'Name', 'quay.io/bogus');
                } else {
                    typeAndSelectCustomSearchFilterValue('Image', 'quay.io/bogus');
                }
            }).should((xhr) => {
                // On switching views, all filters, including the defaults should be cleared
                const requestQuery = xhr.request.body.variables.query.toLowerCase();
                // The request should complete with only a filter for images without cves
                expect(requestQuery).to.contain('image cve count:0');
                expect(requestQuery).to.contain('image:r/quay.io/bogus');
            });

            interactAndWaitForImageList(() => {
                // switching back to the "with cves" view should clear existing filters
                // and reapply the default filters
                changeObservedCveViewingMode('Image vulnerabilities');
                // Check that default filters are present
                cy.get(selectors.filterChipGroupItem(severityChip, 'Critical'));
                cy.get(selectors.filterChipGroupItem(severityChip, 'Important'));
                cy.get(selectors.filterChipGroupItem(cveStatusChip, 'Fixable'));
                // check that the local applied filter is not present
                cy.get(selectors.filterChipGroupItem(imageNameChip, 'quay.io/bogus')).should(
                    'not.exist'
                );
            }).should((xhr) => {
                const requestQuery = xhr.request.body.variables.query.toLowerCase();
                expect(requestQuery).to.contain('vulnerability state');
                expect(requestQuery).to.contain(
                    'severity:critical_vulnerability_severity,important_vulnerability_severity'
                );
                expect(requestQuery).to.contain('fixable:true');
                expect(requestQuery).not.to.contain('image cve count');
                expect(requestQuery).not.to.contain('image:r/quay.io/bogus');
            });
        });
    });
});
