import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import { graphql } from '../../../constants/apiEndpoints';
import {
    applyLocalSeverityFilters,
    selectEntityTab,
    visitWorkloadCveOverview,
    typeAndSelectCustomSearchFilterValue,
    changeObservedCveViewingMode,
} from './WorkloadCves.helpers';
import { selectors } from './WorkloadCves.selectors';
import { interactAndWaitForResponses } from '../../../helpers/request';

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
            cy.get(selectors.filterChipGroupItem('Severity', 'Critical'));
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

    describe.only('Images without CVEs view tests', () => {
        const cveListRouteMatcherMap = {
            getImageCVEList: {
                method: 'POST',
                url: `/api/graphql?opname=getImageCVEList`,
                times: 1,
            },
        };

        const imageListRouteMatcherMap = {
            getImageList: {
                method: 'POST',
                url: `/api/graphql?opname=getImageList`,
                times: 1,
            },
        };

        const deploymentListRouteMatcherMap = {
            getDeploymentList: {
                method: 'POST',
                url: `/api/graphql?opname=getDeploymentList`,
                times: 1,
            },
        };

        beforeEach(function () {
            if (!hasFeatureFlag('ROX_VULN_MGMT_NO_CVES_VIEW')) {
                this.skip();
            }
        });

        it('should remove cve-related UI elements when viewing the "without cves" view', () => {
            visitWorkloadCveOverview();

            // eslint-disable-next-line no-unused-vars
            const riskPriorityHeader = 'th:contains("Risk priority")';
            // eslint-disable-next-line no-unused-vars
            const cvesBySeverityHeader = 'th:contains("CVEs by severity")';
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
            interactAndWaitForResponses(() => {
                selectEntityTab('Image');
            }, imageListRouteMatcherMap);

            assertCveElementsArePresent();

            // Visit the Images tab
            interactAndWaitForResponses(() => {
                selectEntityTab('Deployment');
            }, deploymentListRouteMatcherMap);

            assertCveElementsArePresent();

            // Switch to the "without cves" view, we should stay on the deployments tab
            interactAndWaitForResponses(() => {
                changeObservedCveViewingMode('Images without vulnerabilities');
            }, deploymentListRouteMatcherMap);

            assertCveElementsAreNotPresent();

            // Visit the Images tab
            interactAndWaitForResponses(() => {
                selectEntityTab('Image');
            }, imageListRouteMatcherMap);

            assertCveElementsAreNotPresent();
        });

        it('should apply the correct filters when switching between "with cves" and "without cves" views', () => {
            // Since we want to test the behavior of the default filters with the two cve views, we
            // do not clear them by default in this case
            visitWorkloadCveOverview({ clearFiltersOnVisit: false });

            interactAndWaitForResponses(() => {
                // Add a local filter
                typeAndSelectCustomSearchFilterValue('Image', 'quay.io/bogus');

                // Check that default filters and the local filter are present
                cy.get(selectors.filterChipGroupItem('Severity', 'Critical'));
                cy.get(selectors.filterChipGroupItem('Severity', 'Important'));
                cy.get(selectors.filterChipGroupItem('CVE status', 'Fixable'));
                cy.get(selectors.filterChipGroupItem('Image', 'quay.io/bogus'));
            }, cveListRouteMatcherMap).should((xhr) => {
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

            interactAndWaitForResponses(() => {
                // Change to the "without cves" view, note that since we are currently on the
                // CVE tab, we should automatically switch to the Image tab
                changeObservedCveViewingMode('Images without vulnerabilities');

                // Filters should be cleared
                cy.get(selectors.filterChipGroup).should('not.exist');
            }, imageListRouteMatcherMap).should((xhr) => {
                // On switching views, all filters, including the defaults should be cleared
                const requestQuery = xhr.request.body.variables.query.toLowerCase();
                // The request should complete with only a filter for images without cves
                expect(requestQuery).to.equal('image cve count:0');
            });

            interactAndWaitForResponses(() => {
                // Apply a filter in the "without cves" view
                typeAndSelectCustomSearchFilterValue('Image', 'quay.io/bogus');
            }, imageListRouteMatcherMap).should((xhr) => {
                // On switching views, all filters, including the defaults should be cleared
                const requestQuery = xhr.request.body.variables.query.toLowerCase();
                // The request should complete with only a filter for images without cves
                expect(requestQuery).to.contain('image cve count:0');
                expect(requestQuery).to.contain('image:r/quay.io/bogus');
            });

            interactAndWaitForResponses(() => {
                // switching back to the "with cves" view should clear existing filters
                // and reapply the default filters
                changeObservedCveViewingMode('Image vulnerabilities');
                // Check that default filters are present
                cy.get(selectors.filterChipGroupItem('Severity', 'Critical'));
                cy.get(selectors.filterChipGroupItem('Severity', 'Important'));
                cy.get(selectors.filterChipGroupItem('CVE status', 'Fixable'));
                // check that the local applied filter is not present
                cy.get(selectors.filterChipGroupItem('Image', 'quay.io/bogus')).should('not.exist');
            }, imageListRouteMatcherMap).should((xhr) => {
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
