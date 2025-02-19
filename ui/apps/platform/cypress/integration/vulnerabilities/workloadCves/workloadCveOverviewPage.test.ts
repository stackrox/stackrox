import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import { visitFromHorizontalNav, visitFromHorizontalNavExpandable } from '../../../helpers/nav';
import { graphql } from '../../../constants/apiEndpoints';
import {
    applyDefaultFilters,
    applyLocalSeverityFilters,
    interactAndWaitForImageList,
    interactAndWaitForDeploymentList,
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
    getRouteMatcherMapForGraphQL,
    expectRequestedSort,
    interceptAndWatchRequests,
    interceptAndOverridePermissions,
    interceptAndOverrideFeatureFlags,
} from '../../../helpers/request';

const visitFromMoreViewsDropdown = visitFromHorizontalNavExpandable('More Views');

describe('Workload CVE overview page tests', () => {
    withAuth();

    it('should satisfy initial page load defaults', () => {
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
            const filterChipGroupName = 'CVE severity';
            cy.get(selectors.filterChipGroupItem(filterChipGroupName, 'Critical'));
            cy.get(selectors.filterChipGroupItems(filterChipGroupName)).should('have.lengthOf', 1);

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

    it('should apply the correct baseline filters when switching between built in views using the user-workload based template', function () {
        if (!hasFeatureFlag('ROX_PLATFORM_CVE_SPLIT')) {
            this.skip();
        }

        interceptAndWatchRequests(
            getRouteMatcherMapForGraphQL(['getImageCVEList', 'getImageList'])
        ).then(({ waitForRequests, waitAndYieldRequestBodyVariables }) => {
            visitWorkloadCveOverview();
            waitForRequests(['getImageCVEList']); // Wait for the initial request to complete
            applyDefaultFilters(['Critical', 'Important'], ['Fixable']); // Set the default filters to none to prevent multiple requests on each page visit
            waitForRequests(['getImageCVEList']); // Wait for the third request after the filters have been changed to complete

            // Test the 'User Workloads' view
            visitFromHorizontalNav('User Workloads');
            waitAndYieldRequestBodyVariables(['getImageCVEList']).then(({ query }) => {
                const requestQuery = query.toLowerCase();
                expect(requestQuery).to.contain('vulnerability state:observed');
                expect(requestQuery).to.contain('platform component:false');
                expect(requestQuery).not.to.contain('image cve count');
            });

            // Test the 'Platform' view which is the default in e2e tests
            visitFromHorizontalNav('Platform');
            waitAndYieldRequestBodyVariables(['getImageCVEList']).then(({ query }) => {
                const requestQuery = query.toLowerCase();
                expect(requestQuery).to.contain('vulnerability state:observed');
                expect(requestQuery).to.contain('platform component:true');
                expect(requestQuery).not.to.contain('image cve count');
            });

            // Test the 'All vulnerable images' view
            visitFromHorizontalNavExpandable('More Views')('All vulnerable images');
            waitAndYieldRequestBodyVariables(['getImageCVEList']).then(({ query }) => {
                const requestQuery = query.toLowerCase();
                expect(requestQuery).to.contain('vulnerability state:observed');
                expect(requestQuery).to.contain('platform component:true,false,-');
                expect(requestQuery).not.to.contain('image cve count');
            });

            // Test the 'Inactive images' view
            visitFromHorizontalNavExpandable('All vulnerable images')('Inactive images');
            waitAndYieldRequestBodyVariables(['getImageCVEList']).then(({ query }) => {
                const requestQuery = query.toLowerCase();
                expect(requestQuery).to.contain('vulnerability state:observed');
                expect(requestQuery).to.contain('platform component:-');
                expect(requestQuery).not.to.contain('image cve count');
            });

            // Test the 'Images without CVEs' view
            visitFromHorizontalNavExpandable('Inactive images')('Images without CVEs');
            waitAndYieldRequestBodyVariables(['getImageList']).then(({ query }) => {
                const requestQuery = query.toLowerCase();
                expect(requestQuery).not.to.contain('platform component:observed');
                expect(requestQuery).not.to.contain('vulnerability state');
                expect(requestQuery).to.contain('image cve count:0');
            });
        });
    });

    describe('Column management tests', () => {
        it('should allow the user to hide and show columns on the CVE tab', () => {
            visitWorkloadCveOverview();
            verifyColumnManagement({ tableSelector: 'table' });
        });

        it('should allow the user to hide and show columns on the Images tab', () => {
            visitWorkloadCveOverview();
            selectEntityTab('Image');
            verifyColumnManagement({ tableSelector: 'table' });
        });

        it('should allow the user to hide and show columns on the Deployment tab', () => {
            visitWorkloadCveOverview();
            selectEntityTab('Deployment');
            verifyColumnManagement({ tableSelector: 'table' });
        });
    });

    describe('SBOM generation tests', () => {
        const rowMenuSbomModalButton = 'button[role="menuitem"]:contains("Generate SBOM")';
        const generateSbomButton = '[role="dialog"] button:contains("Generate SBOM")';

        before(function () {
            if (!hasFeatureFlag('ROX_SBOM_GENERATION')) {
                this.skip();
            }
        });

        it('should hide the SBOM generation menu item when the user does not have write access to the Image resource', () => {
            interceptAndOverridePermissions({ Image: 'READ_ACCESS' });

            visitWorkloadCveOverview();
            selectEntityTab('Image');
            openTableRowActionMenu(selectors.firstTableRow);

            cy.get(rowMenuSbomModalButton).should('not.exist');
        });

        it('should disable the SBOM generation button when Scanner V4 is not enabled', () => {
            interceptAndOverrideFeatureFlags({ ROX_SCANNER_V4: false });

            visitWorkloadCveOverview();
            selectEntityTab('Image');
            openTableRowActionMenu(selectors.firstTableRow);

            cy.get(rowMenuSbomModalButton).should('have.attr', 'aria-disabled', 'true');
        });

        it('should trigger a download of the image SBOM via confirmation modal', function () {
            if (!hasFeatureFlag('ROX_SCANNER_V4')) {
                this.skip();
            }

            visitWorkloadCveOverview();
            selectEntityTab('Image');

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
        it('should remove cve-related UI elements when viewing the "without cves" view', function () {
            if (!hasFeatureFlag('ROX_PLATFORM_CVE_SPLIT')) {
                this.skip();
            }

            visitWorkloadCveOverview();

            const cvesBySeverityHeader = 'th:contains("CVEs by severity")';
            const prioritizeByNamespaceButton = 'a:contains("Prioritize by namespace view")';
            const defaultFiltersButton = 'button:contains("Default filters")';

            function assertCveElementsArePresent() {
                cy.get(cvesBySeverityHeader);
                cy.get(prioritizeByNamespaceButton);
                cy.get(defaultFiltersButton);
                cy.get(selectors.severityDropdown);
                cy.get(selectors.fixabilityDropdown);
            }

            function assertCveElementsAreNotPresent() {
                cy.get(cvesBySeverityHeader).should('not.exist');
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

        it('should default to multi-severity sort and keep in sync with applied filters', () => {
            interceptAndWatchRequests(getRouteMatcherMapForGraphQL(['getImageCVEList'])).then(
                ({ waitAndYieldRequestBodyVariables }) => {
                    visitWorkloadCveOverview({ clearFiltersOnVisit: false });

                    // Check the default sort
                    waitAndYieldRequestBodyVariables().then(
                        expectRequestedSort([
                            { field: 'Critical Severity Count', reversed: true },
                            { field: 'Important Severity Count', reversed: true },
                        ])
                    );

                    // Check that adding a severity filter changes the sort
                    applyLocalSeverityFilters('Moderate');
                    waitAndYieldRequestBodyVariables().then(
                        expectRequestedSort([
                            { field: 'Critical Severity Count', reversed: true },
                            { field: 'Important Severity Count', reversed: true },
                            { field: 'Moderate Severity Count', reversed: true },
                        ])
                    );

                    // Check that the severity sort is reversible
                    sortByTableHeader('Images by severity');
                    waitAndYieldRequestBodyVariables().then(
                        expectRequestedSort([
                            { field: 'Critical Severity Count', reversed: false },
                            { field: 'Important Severity Count', reversed: false },
                            { field: 'Moderate Severity Count', reversed: false },
                        ])
                    );

                    // Check that sorting by another column works as intended
                    sortByTableHeader('CVE');
                    waitAndYieldRequestBodyVariables().then(
                        expectRequestedSort({ field: 'CVE', reversed: true })
                    );

                    // Check that changing the severity filter when a non-severity sort is applied
                    // maintains the current sort
                    applyLocalSeverityFilters('Low');
                    waitAndYieldRequestBodyVariables().then(
                        expectRequestedSort({ field: 'CVE', reversed: true })
                    );

                    // Check that visiting via a direct link that includes a severity filter maintains
                    // the correct sort
                    visitWorkloadCveOverview({
                        clearFiltersOnVisit: false,
                        urlSearch: '?s[SEVERITY][0]=Important',
                    });
                    waitAndYieldRequestBodyVariables().then(
                        expectRequestedSort([{ field: 'Important Severity Count', reversed: true }])
                    );
                }
            );
        });
    });
});
