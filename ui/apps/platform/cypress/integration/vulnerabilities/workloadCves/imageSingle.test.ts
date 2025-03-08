import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import {
    getRouteMatcherMapForGraphQL,
    interactAndWaitForResponses,
    interceptAndOverrideFeatureFlags,
    interceptAndOverridePermissions,
    interceptAndWatchRequests,
} from '../../../helpers/request';
import {
    changePerPageOption,
    sortByTableHeader,
    verifyColumnManagement,
} from '../../../helpers/tableHelpers';

import { selectors as vulnSelectors } from '../vulnerabilities.selectors';
import {
    applyLocalSeverityFilters,
    typeAndEnterSearchFilterValue,
    selectEntityTab,
    visitWorkloadCveOverview,
} from './WorkloadCves.helpers';
import { selectors } from './WorkloadCves.selectors';

describe('Workload CVE Image Single page', () => {
    withAuth();

    function visitFirstImage(): Promise<string> {
        visitWorkloadCveOverview();

        selectEntityTab('Image');

        // Ensure the data in the table has settled
        cy.get(selectors.isUpdatingTable).should('not.exist');

        return cy.get('tbody tr td[data-label="Image"] a').then(([$imageLink]) => {
            const imageName = $imageLink.innerText.replace('\n', '');
            cy.wrap($imageLink).click();
            cy.get('h1').contains(imageName);
            return Promise.resolve(imageName);
        });
    }

    it('should contain the correct search filters in the toolbar', () => {
        visitFirstImage();

        // Check that only applicable resource menu items are present in the toolbar
        cy.get(selectors.searchEntityDropdown).click();
        cy.get(selectors.searchEntityMenuItem).contains('Image');
        cy.get(selectors.searchEntityMenuItem).contains('Image component');
        cy.get(selectors.searchEntityDropdown).click();
    });

    it('should display consistent data between the cards and the table test', () => {
        visitFirstImage();

        // Check that the CVEs by severity totals in the card match the number in the "results found" text
        const cardSelector = vulnSelectors.summaryCard('CVEs by severity');
        cy.get(
            [
                `${cardSelector} span.pf-v5-c-icon:contains("Critical") ~ p`,
                `${cardSelector} span.pf-v5-c-icon:contains("Important") ~ p`,
                `${cardSelector} span.pf-v5-c-icon:contains("Moderate") ~ p`,
                `${cardSelector} span.pf-v5-c-icon:contains("Low") ~ p`,
            ].join(',')
        ).then(($severityTotals) => {
            const severityTotal = $severityTotals.toArray().reduce((acc, $el) => {
                const count = acc + parseInt($el.innerText.replace(/\D/g, ''), 10);
                return Number.isNaN(count) ? acc : count;
            }, 0);
            const plural = severityTotal === 1 ? '' : 's';
            cy.get(`*:contains(${severityTotal} result${plural} found)`);
        });

        // Check that the CVEs by status totals in the card match the number in the "results found" text
        const fixStatusCardSelector = vulnSelectors.summaryCard('CVEs by status');
        cy.get(
            [
                `${fixStatusCardSelector} p:contains('with available fixes')`,
                `${fixStatusCardSelector} p:contains("without fixes")`,
            ].join(',')
        ).then(($statusTotals) => {
            const statusTotal = $statusTotals.toArray().reduce((acc, $el) => {
                const count = acc + parseInt($el.innerText.replace(/\D/g, ''), 10);
                return Number.isNaN(count) ? 0 : count;
            }, 0);

            const plural = statusTotal === 1 ? '' : 's';
            cy.get(`*:contains(${statusTotal} result${plural} found)`);
        });
    });

    it('should correctly apply a severity filter', () => {
        visitFirstImage();
        // Check that no severities are hidden by default
        cy.get(vulnSelectors.summaryCard('CVEs by severity'))
            .find('p')
            .contains(new RegExp('(Critical|Important|Moderate|Low) hidden'))
            .should('not.exist');

        const severityFilter = 'Critical';

        applyLocalSeverityFilters(severityFilter);

        // Check that summary card severities are hidden correctly
        cy.get(`*:contains("Critical hidden")`).should('not.exist');
        cy.get(`*:contains("Important hidden")`);
        cy.get(`*:contains("Moderate hidden")`);
        cy.get(`*:contains("Low hidden")`);

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
                typeAndEnterSearchFilterValue('CVE', 'Name', cveName);
                // Check that the header above the table shows only one result
                cy.get(`*:contains("1 result found")`);
                // Check that the only row in the table has the correct CVE name
                cy.get(`tbody tr td[data-label="CVE"]`).should('have.length', 1);
                cy.get(`tbody tr td[data-label="CVE"]:contains("${cveName}")`);
            });
    });

    // Verifies that the data returned by the server is not duplicated due to Apollo client cache issues
    // see: https://issues.redhat.com/browse/ROX-24254
    //      https://github.com/stackrox/stackrox/pull/6156
    it('should display nested component data correctly when processed via apollo client', () => {
        const opname = 'getCVEsForImage';
        const routeMatcherMap = getRouteMatcherMapForGraphQL([opname]);
        const body = {
            data: {
                image: {
                    id: 'sha256:010fec71f42f4b5e65f3f56f10af94a7c05c9c271a9bbc3026684ba170698cb5',
                    name: {
                        registry: 'quay.io',
                        remote: 'openshift-release-dev/ocp-v4.0-art-dev',
                        tag: '',
                        __typename: 'ImageName',
                    },
                    metadata: {
                        v1: {
                            layers: [
                                {
                                    instruction: 'ADD',
                                    value: 'file:091e888311e2628528312ffc60e27702fe04b23f8e4c95b456c16a967cdd89e0 in /',
                                    __typename: 'ImageLayer',
                                },
                            ],
                            __typename: 'V1Metadata',
                        },
                        __typename: 'ImageMetadata',
                    },
                    __typename: 'Image',
                    imageCVECountBySeverity: {
                        low: {
                            total: 0,
                            fixable: 0,
                            __typename: 'ResourceCountByFixability',
                        },
                        moderate: {
                            total: 1,
                            fixable: 1,
                            __typename: 'ResourceCountByFixability',
                        },
                        important: {
                            total: 0,
                            fixable: 0,
                            __typename: 'ResourceCountByFixability',
                        },
                        critical: {
                            total: 0,
                            fixable: 0,
                            __typename: 'ResourceCountByFixability',
                        },
                        __typename: 'ResourceCountByCVESeverity',
                    },
                    imageVulnerabilities: [
                        {
                            severity: 'MODERATE_VULNERABILITY_SEVERITY',
                            cve: '[CYPRESS-MOCKED] CVE-2023-44487',
                            summary: 'HTTP/2 Stream Cancellation Attack',
                            cvss: 5.300000190734863,
                            scoreVersion: 'V3',
                            discoveredAtImage: '2024-04-03T19:44:55.837891332Z',
                            pendingExceptionCount: 0,
                            imageComponents: [
                                {
                                    name: 'golang.org/x/net',
                                    version: 'v0.13.0',
                                    location: 'usr/bin/cluster-samples-operator-watch',
                                    source: 'GO',
                                    layerIndex: 0,
                                    imageVulnerabilities: [
                                        {
                                            vulnerabilityId: 'CVE-2023-44487#rhel:9',
                                            severity: 'MODERATE_VULNERABILITY_SEVERITY',
                                            fixedByVersion: '0.17.0',
                                            pendingExceptionCount: 0,
                                            __typename: 'ImageVulnerability',
                                        },
                                    ],
                                    __typename: 'ImageComponent',
                                },
                                {
                                    name: 'google.golang.org/grpc',
                                    version: 'v1.54.0',
                                    location: 'usr/bin/cluster-samples-operator-watch',
                                    source: 'GO',
                                    layerIndex: 0,
                                    imageVulnerabilities: [
                                        {
                                            vulnerabilityId: 'CVE-2023-44487#rhel:9',
                                            severity: 'MODERATE_VULNERABILITY_SEVERITY',
                                            fixedByVersion: '1.56.3',
                                            pendingExceptionCount: 0,
                                            __typename: 'ImageVulnerability',
                                        },
                                    ],
                                    __typename: 'ImageComponent',
                                },
                                {
                                    name: 'openshift4/ose-cluster-samples-rhel9-operator',
                                    version: 'v4.15.0-202401261531.p0.gd546ec2.assembly.stream',
                                    location:
                                        'root/buildinfo/Dockerfile-openshift-ose-cluster-samples-rhel9-operator-v4.15.0-202401261531.p0.gd546ec2.assembly.stream',
                                    source: 'OS',
                                    layerIndex: 0,
                                    imageVulnerabilities: [
                                        {
                                            vulnerabilityId: 'CVE-2023-44487#rhel:9',
                                            severity: 'MODERATE_VULNERABILITY_SEVERITY',
                                            fixedByVersion:
                                                'v4.15.0-202404031310.p0.gbf845b5.assembly.stream.el9',
                                            pendingExceptionCount: 0,
                                            __typename: 'ImageVulnerability',
                                        },
                                    ],
                                    __typename: 'ImageComponent',
                                },
                            ],
                            __typename: 'ImageVulnerability',
                        },
                    ],
                },
            },
        };

        const staticResponseMap = { [opname]: { body } };

        interactAndWaitForResponses(
            () => {
                visitFirstImage();
            },
            routeMatcherMap,
            staticResponseMap
        );

        cy.get(vulnSelectors.expandRowButton).click();

        const fixedInCellSelector = `table td[data-label="CVE fixed in"]`;
        const components = body.data.image.imageVulnerabilities[0].imageComponents;

        components.forEach((component, index) => {
            cy.get(fixedInCellSelector)
                .eq(index)
                .contains(component.imageVulnerabilities[0].fixedByVersion);
        });
    });

    // See case 03985920 and ROX-27344 for more details
    it('should receive consistent CVE counts when sorting and paginating the table', () => {
        const opname = 'getCVEsForImage';
        const routeMatcherMap = getRouteMatcherMapForGraphQL([opname]);

        // Captures the initial CVE count and the initial query sent when visiting the image details page
        // and uses these values as a basis of comparison on subsequent requests
        function createAssertion(initialCount: number, initialQuery: string) {
            return function (interception) {
                expect(interception.request.body.variables.query).to.equal(initialQuery);
                expect(interception.response.body.data.image.imageVulnerabilityCount).to.equal(
                    initialCount
                );
            };
        }

        // Test count stability with no filters applied
        interceptAndWatchRequests(routeMatcherMap).then(({ waitForRequests }) => {
            visitFirstImage();
            waitForRequests()
                .then(({ request, response }) => ({
                    assertCveCountsUnchanged: createAssertion(
                        response.body.data.image.imageVulnerabilityCount,
                        request.body.variables.query
                    ),
                }))
                .then(({ assertCveCountsUnchanged }) => {
                    // Check the initial sort request
                    sortByTableHeader('CVE');
                    waitForRequests().then(assertCveCountsUnchanged);

                    // Check the initial perPage change request
                    changePerPageOption(50);
                    waitForRequests().then(assertCveCountsUnchanged);

                    // Test another sort back-and-forth
                    sortByTableHeader('CVE severity');
                    waitForRequests().then(assertCveCountsUnchanged);
                    sortByTableHeader('CVE severity');
                    waitForRequests().then(assertCveCountsUnchanged);
                    sortByTableHeader('CVE severity');
                    waitForRequests().then(assertCveCountsUnchanged);

                    // Test changing back to the original pagination
                    changePerPageOption(20);
                    waitForRequests().then(assertCveCountsUnchanged);

                    // Test a pagination change after returning to the default
                    changePerPageOption(10);
                    waitForRequests().then(assertCveCountsUnchanged);

                    // Test sorting by a column already used as a sort *again*
                    sortByTableHeader('CVE severity');
                    waitForRequests().then(assertCveCountsUnchanged);

                    // Test sorting on the only remaining untested column by rapidly changing
                    // the column value without waiting for a response
                    sortByTableHeader('CVSS');
                    sortByTableHeader('CVSS');
                    sortByTableHeader('CVSS');
                    sortByTableHeader('CVSS');
                    waitForRequests().then(assertCveCountsUnchanged);
                    waitForRequests().then(assertCveCountsUnchanged);
                    waitForRequests().then(assertCveCountsUnchanged);
                    waitForRequests().then(assertCveCountsUnchanged);
                });
        });
    });

    describe('Column management tests', () => {
        it('should allow the user to hide and show columns on the CVE table', () => {
            visitFirstImage();
            verifyColumnManagement({ tableSelector: 'table' });
        });
    });

    describe('SBOM generation tests', () => {
        const headerSbomModalButton = 'section:has(h1) button:contains("Generate SBOM")';
        const generateSbomButton = '[role="dialog"] button:contains("Generate SBOM")';

        before(function () {
            if (!hasFeatureFlag('ROX_SBOM_GENERATION')) {
                this.skip();
            }
        });

        it('should hide the SBOM generation button when the user does not have write access to the Image resource', () => {
            interceptAndOverridePermissions({ Image: 'READ_ACCESS' });

            visitFirstImage();

            cy.get(headerSbomModalButton).should('not.exist');
        });

        it('should disable the SBOM generation button when Scanner V4 is not enabled', () => {
            interceptAndOverrideFeatureFlags({ ROX_SCANNER_V4: false });

            visitFirstImage();

            cy.get(headerSbomModalButton).should('have.attr', 'aria-disabled', 'true');
        });

        it('should trigger a download of the image SBOM via confirmation modal', function () {
            if (!hasFeatureFlag('ROX_SCANNER_V4')) {
                this.skip();
            }

            visitFirstImage().then((imageFullName) => {
                cy.get(headerSbomModalButton).click();
                cy.get(selectors.generateSbomModal).contains(imageFullName);
                cy.get(generateSbomButton).click();
                cy.get(':contains("Generating, please do not navigate away from this modal")');
                cy.get(':contains("Software Bill of Materials (SBOM) generated successfully")');
            });
        });
    });
});
