import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import { visitFromHorizontalNav, visitFromHorizontalNavExpandable } from '../../../helpers/nav';
import {
    applyDefaultFilters,
    interactAndWaitForDeploymentList,
    interactAndWaitForImageList,
    mockSbomGenerationRequest,
    selectEntityTab,
    visitWorkloadCveOverview,
} from './WorkloadCves.helpers';
import { selectors } from './WorkloadCves.selectors';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';
import {
    openTableRowActionMenu,
    sortByTableHeader,
    verifyColumnManagement,
} from '../../../helpers/tableHelpers';
import {
    getRouteMatcherMapForVulnMgmt,
    interceptAndOverrideFeatureFlags,
    interceptAndOverridePermissions,
    interceptAndWatchRequests,
} from '../../../helpers/request';
import { addCheckboxSelectFilter } from '../../../helpers/compoundFilters';
import { visit } from '../../../helpers/visit';

const visitFromMoreViewsDropdown = visitFromHorizontalNavExpandable('More Views');

describe('Workload CVE overview page tests', () => {
    withAuth();

    it('should satisfy initial page load defaults', () => {
        cy.spyTelemetry();
        visitWorkloadCveOverview();

        // TODO Test that the default tab is set to "Observed"

        // Check that the CVE entity toggle is selected and Image/Deployment are disabled
        cy.get(vulnSelectors.entityTypeToggleItem('CVE')).should(
            'have.attr',
            'aria-pressed',
            'true'
        );
        cy.get(vulnSelectors.entityTypeToggleItem('Image')).should(
            'not.have.attr',
            'aria-pressed',
            'true'
        );
        cy.get(vulnSelectors.entityTypeToggleItem('Deployment')).should(
            'not.have.attr',
            'aria-pressed',
            'true'
        );
        cy.getTelemetryEvents().should((telemetryEvents) => {
            // TODO - This is not working as expected. We should only have one page view event, but the page view event is being emitted twice.
            expect(telemetryEvents.page).to.have.length(2);

            expect(telemetryEvents.page[0].properties.path).to.contain(
                '/main/vulnerabilities/platform'
            );

            expect(telemetryEvents.track).to.have.length(1);
            expect(telemetryEvents.track[0].event).to.equal('Workload CVE Entity Context View');
            expect(telemetryEvents.track[0].properties.type).to.equal('CVE');
            expect(telemetryEvents.track[0].properties.page).to.equal('Overview');
        });
    });

    it('should apply the selected severity tab to every entity tab', () => {
        visitWorkloadCveOverview();

        cy.intercept(
            { method: 'GET', url: '/api/v2/vulnmgmt/images?*' },
            { body: { items: [], totalCount: 0 } }
        ).as('getSeverityTabImages');
        cy.intercept(
            { method: 'GET', url: '/api/v2/vulnmgmt/deployments?*' },
            { body: { items: [], totalCount: 0 } }
        ).as('getSeverityTabDeployments');

        function expectImportantSeverity(url: string) {
            const query = new URL(url, 'http://localhost').searchParams.get('query') ?? '';
            expect(decodeURIComponent(query)).to.contain(
                'Severity:IMPORTANT_VULNERABILITY_SEVERITY'
            );
        }

        cy.get('[role="tab"]').contains('Important').click();
        cy.wait('@getVulnMgmtCVEs').then(({ request }) => {
            expectImportantSeverity(request.url);
        });

        selectEntityTab('Image');
        cy.wait('@getSeverityTabImages').then(({ request }) => {
            expectImportantSeverity(request.url);
        });

        selectEntityTab('Deployment');
        cy.wait('@getSeverityTabDeployments').then(({ request }) => {
            expectImportantSeverity(request.url);
        });
    });

    it('should apply the correct baseline filters when switching between built in views using the user-workload based template', () => {
        interceptAndWatchRequests(getRouteMatcherMapForVulnMgmt()).then(
            ({ waitForRequests, waitAndYieldRequestQueryParameters }) => {
                // Test the 'Platform' view which is the default in e2e tests
                visit('/main/vulnerabilities/platform/');
                waitAndYieldRequestQueryParameters(['getVulnMgmtCVEs']).then((query) => {
                    const requestQuery = query.toLowerCase();
                    expect(requestQuery).to.contain('vulnerability state:observed');
                    expect(requestQuery).to.contain('platform component:true');
                    expect(requestQuery).not.to.contain('image cve count');
                });

                applyDefaultFilters(['Critical', 'Important'], ['Fixable']);
                waitForRequests(['getVulnMgmtCVEs', 'getVulnMgmtCounts']);

                // Test the 'User Workloads' view
                visitFromHorizontalNav('User Workloads');
                waitAndYieldRequestQueryParameters(['getVulnMgmtCVEs']).then((query) => {
                    const requestQuery = query.toLowerCase();
                    expect(requestQuery).to.contain('vulnerability state:observed');
                    expect(requestQuery).to.contain('platform component:false');
                    expect(requestQuery).not.to.contain('image cve count');
                });

                // Test the 'All vulnerable images' view
                visitFromHorizontalNavExpandable('More Views')('All vulnerable images');
                waitAndYieldRequestQueryParameters(['getVulnMgmtCVEs']).then((query) => {
                    const requestQuery = query.toLowerCase();
                    expect(requestQuery).to.contain('vulnerability state:observed');
                    expect(requestQuery).not.to.contain('platform component');
                    expect(requestQuery).not.to.contain('image cve count');
                });

                // Test the 'Inactive images' view
                visitFromHorizontalNavExpandable('All vulnerable images')('Inactive images');
                waitAndYieldRequestQueryParameters(['getVulnMgmtCVEs']).then((query) => {
                    const requestQuery = query.toLowerCase();
                    expect(requestQuery).to.contain('vulnerability state:observed');
                    expect(requestQuery).to.contain('platform component:-');
                    expect(requestQuery).not.to.contain('image cve count');
                });

                // Test the 'Images without CVEs' view
                visitFromHorizontalNavExpandable('Inactive images')('Images without CVEs');
                waitAndYieldRequestQueryParameters(['getVulnMgmtImages']).then((query) => {
                    const requestQuery = query.toLowerCase();
                    expect(requestQuery).not.to.contain('platform component:observed');
                    expect(requestQuery).not.to.contain('vulnerability state');
                    expect(requestQuery).to.contain('image cve count:0');
                });
            }
        );
    });

    describe('Column management tests', () => {
        it('should allow the user to hide and show columns on the CVE tab', () => {
            visitWorkloadCveOverview();
            verifyColumnManagement({ tableSelector: 'table' });
        });

        it('should allow the user to hide and show columns on the Images tab', () => {
            visitWorkloadCveOverview();
            interactAndWaitForImageList(() => selectEntityTab('Image'));
            verifyColumnManagement({ tableSelector: 'table' });
        });

        it('should allow the user to hide and show columns on the Deployment tab', () => {
            visitWorkloadCveOverview();
            interactAndWaitForDeploymentList(() => selectEntityTab('Deployment'));
            verifyColumnManagement({ tableSelector: 'table' });
        });
    });

    describe('SBOM generation tests', () => {
        const rowMenuSbomModalButton = 'button[role="menuitem"]:contains("Generate SBOM")';
        const generateSbomButton = '[role="dialog"] button:contains("Generate SBOM")';

        it('should hide the SBOM generation menu item when the user does not have write access to the Image resource', () => {
            interceptAndOverridePermissions({ Image: 'READ_ACCESS' });

            visitWorkloadCveOverview();
            interactAndWaitForImageList(() => selectEntityTab('Image'));
            openTableRowActionMenu(selectors.firstTableRow);

            cy.get(rowMenuSbomModalButton).should('not.exist');
        });

        it('should disable the SBOM generation button when Scanner V4 is not enabled', () => {
            interceptAndOverrideFeatureFlags({ ROX_SCANNER_V4: false });

            visitWorkloadCveOverview();
            interactAndWaitForImageList(() => selectEntityTab('Image'));
            openTableRowActionMenu(selectors.firstTableRow);

            cy.get(rowMenuSbomModalButton).should('have.attr', 'aria-disabled', 'true');
        });

        it('should trigger a download of the image SBOM via confirmation modal', function () {
            if (!hasFeatureFlag('ROX_SCANNER_V4')) {
                this.skip();
            }

            visitWorkloadCveOverview();
            interactAndWaitForImageList(() => selectEntityTab('Image'));
            mockSbomGenerationRequest();

            cy.get(selectors.firstTableRow)
                .find('td[data-label="Image"] a')
                .then(($link) => {
                    const imageFullName = $link.text();
                    openTableRowActionMenu(selectors.firstTableRow);

                    cy.get(rowMenuSbomModalButton).click();
                    cy.get(selectors.generateSbomModal).contains(imageFullName);
                    cy.get(generateSbomButton).click();
                    cy.get(':contains("Generating, please do not navigate away from this modal")');
                    cy.get(':contains("Software Bill of Materials (SBOM) generated successfully")');
                });
        });
    });

    describe('Images without CVEs view tests', () => {
        it('should remove cve-related UI elements when viewing the "without cves" view', () => {
            visitWorkloadCveOverview();

            const matchingCveCountHeader = 'th:contains("Matching CVEs")';
            const severityTabs = '[role="tablist"]:contains("Critical")';
            const namespaceViewLink = 'a:contains("Namespace view")';
            const defaultFiltersButton = 'button:contains("Default filters")';

            function assertCveElementsArePresent() {
                cy.get(matchingCveCountHeader);
                cy.get(namespaceViewLink);
                cy.get(defaultFiltersButton);
                cy.get(severityTabs);
                cy.get(selectors.fixabilityDropdown);
            }

            function assertCveElementsAreNotPresent() {
                cy.get(matchingCveCountHeader).should('not.be.visible');
                cy.get(namespaceViewLink).should('not.exist');
                cy.get(defaultFiltersButton).should('not.exist');
                cy.get(severityTabs).should('not.exist');
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

            interactAndWaitForImageList(() => {
                visitFromMoreViewsDropdown('Images without CVEs');
            });

            assertCveElementsAreNotPresent();

            // Visit the Deployments tab
            interactAndWaitForDeploymentList(() => {
                selectEntityTab('Deployment');
            });

            assertCveElementsAreNotPresent();
        });

        it('should use the affected-images sort by default and reset it on severity tab change', () => {
            interceptAndWatchRequests(getRouteMatcherMapForVulnMgmt()).then(
                ({ waitAndYieldRequestSort }) => {
                    const basePath = '/main/vulnerabilities/platform/';
                    const defaultSort = {
                        field: 'Image Sha',
                        reversed: true,
                        aggregateBy: { aggregateFunc: 'count', distinct: true },
                    };

                    visit(basePath);
                    waitAndYieldRequestSort(['getVulnMgmtCVEs']).then((sort) => {
                        expect(sort).to.deep.equal(defaultSort);
                    });

                    // Sorting by the CVE column works and replaces the default sort
                    sortByTableHeader('CVE');
                    waitAndYieldRequestSort(['getVulnMgmtCVEs']).then((sort) => {
                        expect(sort).to.deep.equal({ field: 'CVE', reversed: true });
                    });

                    // Changing the severity tab resets the CVE tab sort to the default
                    cy.get('[role="tab"]').contains('Important').click();
                    waitAndYieldRequestSort(['getVulnMgmtCVEs']).then((sort) => {
                        expect(sort).to.deep.equal(defaultSort);
                    });

                    // A direct link with a severity tab still uses the default sort
                    visit(`${basePath}?severityTab=Important`);
                    waitAndYieldRequestSort(['getVulnMgmtCVEs']).then((sort) => {
                        expect(sort).to.deep.equal(defaultSort);
                    });
                }
            );
        });
    });

    describe('Layer type filter tests', () => {
        it('should apply Layer type filter correctly', () => {
            visitWorkloadCveOverview();

            addCheckboxSelectFilter('Image component', 'Layer type', 'Base image');

            // Verify filter chip is applied
            cy.get(selectors.filterLabelGroupForCategory('Image component layer type')).should(
                'be.visible'
            );
            cy.get(
                selectors.filterLabelGroupItem('Image component layer type', 'Base image')
            ).should('be.visible');

            // Verify filtered view label is shown
            cy.get(selectors.filteredViewLabel).should('be.visible');
        });
    });
});
