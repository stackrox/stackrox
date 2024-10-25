import withAuth from '../../../helpers/basicAuth';
import * as filterHelpers from '../../../helpers/compoundFilters';
import {
    expectRequestedPagination,
    expectRequestedQuery,
    expectRequestedSort,
    interactAndWaitForResponses,
    interceptAndWatchRequests,
} from '../../../helpers/request';
import {
    assertOnEachRowForColumn,
    paginateNext,
    paginatePrevious,
    queryTableHeader,
    queryTableSortHeader,
    sortByTableHeader,
} from '../../../helpers/tableHelpers';
import {
    assertCannotFindThePage,
    visitWithStaticResponseForPermissions,
} from '../../../helpers/visit';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';
import {
    applyLocalSeverityFilters,
    applyLocalStatusFilters,
} from '../workloadCves/WorkloadCves.helpers';
import {
    getNodeMetadataOpname,
    getNodeVulnerabilitiesOpname,
    getNodeVulnSummaryOpname,
    routeMatcherMapForNodePage,
    routeMatcherMapForNodes,
    visitFirstNodeFromOverviewPage,
} from './NodeCve.helpers';

const { assertAvailableFilters } = filterHelpers;

const nodeBaseUrl = '/main/vulnerabilities/node-cves/nodes';
const mockNodeId = '1';
const mockNodeName = 'cypress-node-1';

export const staticResponseMapForNodePage = {
    [getNodeMetadataOpname]: {
        fixture: `vulnerabilities/nodeCves/${getNodeMetadataOpname}`,
    },
    [getNodeVulnSummaryOpname]: {
        fixture: `vulnerabilities/nodeCves/${getNodeVulnSummaryOpname}`,
    },
    [getNodeVulnerabilitiesOpname]: {
        fixture: `vulnerabilities/nodeCves/${getNodeVulnerabilitiesOpname}`,
    },
};

const mockNodePageUrl = `${nodeBaseUrl}/${mockNodeId}`;

describe('Node CVEs - Node Detail Page', () => {
    withAuth();

    it('should restrict access to users with insufficient "Node" permission', () => {
        visitWithStaticResponseForPermissions(mockNodePageUrl, {
            body: { resourceToAccess: { Node: 'READ_ACCESS' } },
        });
        assertCannotFindThePage();
    });

    it('should restrict access to users with insufficient "Cluster" permission', () => {
        visitWithStaticResponseForPermissions(mockNodePageUrl, {
            body: { resourceToAccess: { Cluster: 'READ_ACCESS' } },
        });
        assertCannotFindThePage();
    });

    it('should allow access to users with sufficient permissions', () => {
        visitWithStaticResponseForPermissions(
            mockNodePageUrl,
            {
                body: { resourceToAccess: { Node: 'READ_ACCESS', Cluster: 'READ_ACCESS' } },
            },
            routeMatcherMapForNodePage,
            staticResponseMapForNodePage
        );
        cy.get('h1').contains(mockNodeName);
    });

    it('should only show relevant filters for the Node Detail page', () => {
        visitFirstNodeFromOverviewPage();

        assertAvailableFilters({
            CVE: ['Name', 'CVSS', 'Discovered Time'],
            'Node Component': ['Name', 'Version'],
        });
    });

    it('should follow the breadcrumb link to the Node list tab', () => {
        visitFirstNodeFromOverviewPage();

        // clicking the Nodes breadcrumb should navigate to the overview page with the Node tab selected
        interactAndWaitForResponses(() => {
            cy.get('nav[aria-label="Breadcrumb"] a').contains('Nodes').click();
        }, routeMatcherMapForNodes);
        cy.get(`${vulnSelectors.entityTypeToggleItem('CVE')}[aria-pressed=false]`);
        cy.get(`${vulnSelectors.entityTypeToggleItem('Node')}[aria-pressed=true]`);
    });

    it('should link from a CVE in the table to the CVE detail page', () => {
        interactAndWaitForResponses(
            () => {
                visitFirstNodeFromOverviewPage();
            },
            routeMatcherMapForNodePage,
            staticResponseMapForNodePage
        );

        // clicking a CVE name in the list should navigate to a Node CVE details page
        cy.get(`table td[data-label="CVE"]`).first().click();
        cy.get('nav[aria-label="Breadcrumb"] a').contains('Node CVEs');
    });

    it('should sort CVE table columns', () => {
        interceptAndWatchRequests({
            [getNodeVulnerabilitiesOpname]:
                routeMatcherMapForNodePage[getNodeVulnerabilitiesOpname],
        }).then(({ waitForRequests, waitAndYieldRequestBodyVariables }) => {
            visitFirstNodeFromOverviewPage();
            waitForRequests();

            // check sorting of CVE column
            sortByTableHeader('CVE');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedSort({ field: 'CVE', reversed: true })
            );
            sortByTableHeader('CVE');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedSort({ field: 'CVE', reversed: false })
            );

            // check sorting of Top Severity column
            sortByTableHeader('Top severity');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedSort({ field: 'Severity', reversed: true })
            );
            sortByTableHeader('Top severity');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedSort({ field: 'Severity', reversed: false })
            );

            // check sorting of CVE status column
            sortByTableHeader('CVE status');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedSort({ field: 'Fixable', reversed: true })
            );
            sortByTableHeader('CVE status');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedSort({ field: 'Fixable', reversed: false })
            );

            // check sorting of CVSS column
            sortByTableHeader('CVSS');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedSort({ field: 'CVSS', reversed: true })
            );
            sortByTableHeader('CVSS');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedSort({ field: 'CVSS', reversed: false })
            );

            // check that the Affected components column is not sortable
            queryTableHeader('Affected components');
            queryTableSortHeader('Affected components').should('not.exist');
        });
    });

    it('should filter the CVE table', () => {
        interceptAndWatchRequests({
            [getNodeVulnerabilitiesOpname]:
                routeMatcherMapForNodePage[getNodeVulnerabilitiesOpname],
        }).then(({ waitForRequests, waitAndYieldRequestBodyVariables }) => {
            visitFirstNodeFromOverviewPage();
            waitForRequests();

            // filtering by CVE name should only display rows with a matching name
            filterHelpers.addAutocompleteFilter('CVE', 'Name', 'CVE-2021-1234');
            waitAndYieldRequestBodyVariables().then(expectRequestedQuery('CVE:r/CVE-2021-1234'));
            // Do not assert on cell contents as the filter value is mocked
            filterHelpers.clearFilters();
            waitForRequests();

            // filtering by Severity should only display rows with a matching top severity
            applyLocalSeverityFilters('Low');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedQuery('SEVERITY:LOW_VULNERABILITY_SEVERITY')
            );
            assertOnEachRowForColumn('Top severity', (_, cell) => {
                expect(cell.innerText).to.contain('Low');
            });
            filterHelpers.clearFilters();
            waitForRequests();

            // filtering by CVE Status should only display rows with a matching status
            applyLocalStatusFilters('Fixable');
            waitAndYieldRequestBodyVariables().then(expectRequestedQuery('FIXABLE:true'));
            assertOnEachRowForColumn('CVE status', (_, cell) => {
                expect(cell.innerText).to.contain('Fixable');
            });
            filterHelpers.clearFilters();
            waitForRequests();

            // filtering by CVSS should only display rows with a CVSS in range
            filterHelpers.addNumericFilter('CVE', 'CVSS', 'Is less than', 8);
            waitAndYieldRequestBodyVariables().then(expectRequestedQuery('CVSS:<8'));
            assertOnEachRowForColumn('CVSS', (_, cell) => {
                const cvss = parseFloat(cell.innerText.replace(/[^0-9.]/g, ''));
                expect(cvss).to.be.lessThan(8);
            });
            filterHelpers.clearFilters();
            waitForRequests();

            // filtering by Component should only display rows with a nested table containing a matching component
            //   - expand each row
            //   - check that the component name exists in the table
            const componentFilter = 'a';
            filterHelpers.addAutocompleteFilter('Node component', 'Name', componentFilter);
            waitAndYieldRequestBodyVariables().then(
                expectRequestedQuery(`Component:r/${componentFilter}`)
            );
            // scope these assertions to the first parent table so that assertions run in the child tables
            const columnDataLabel = 'Component';
            cy.get(`table`)
                .then(($el) =>
                    $el.find(
                        `table:has(th:contains("${columnDataLabel}")) td[data-label="${columnDataLabel}"]`
                    )
                )
                .then(($cells) => {
                    $cells.each((_, cell) => {
                        expect(cell.innerText).to.contain(componentFilter);
                    });
                });
        });
    });

    it('should correctly paginate the CVE table', () => {
        interceptAndWatchRequests(routeMatcherMapForNodePage, staticResponseMapForNodePage).then(
            ({ waitForRequests, waitAndYieldRequestBodyVariables }) => {
                visitFirstNodeFromOverviewPage();
                waitForRequests();

                paginateNext();
                waitAndYieldRequestBodyVariables([getNodeVulnerabilitiesOpname]).then(
                    expectRequestedPagination({ offset: 20, limit: 20 })
                );

                paginateNext();
                waitAndYieldRequestBodyVariables([getNodeVulnerabilitiesOpname]).then(
                    expectRequestedPagination({ offset: 40, limit: 20 })
                );

                paginatePrevious();
                waitAndYieldRequestBodyVariables([getNodeVulnerabilitiesOpname]).then(
                    expectRequestedPagination({ offset: 20, limit: 20 })
                );

                paginatePrevious();
                waitAndYieldRequestBodyVariables([getNodeVulnerabilitiesOpname]).then(
                    expectRequestedPagination({ offset: 0, limit: 20 })
                );

                // test that applying a filter resets the page to 1
                paginateNext();
                waitForRequests([getNodeVulnerabilitiesOpname]);
                filterHelpers.addAutocompleteFilter('CVE', 'Name', '1');
                waitAndYieldRequestBodyVariables([getNodeVulnerabilitiesOpname]).then(
                    expectRequestedPagination({ offset: 0, limit: 20 })
                );

                // test that applying a sort resets the page to 1
                paginateNext();
                waitForRequests([getNodeVulnerabilitiesOpname]);
                sortByTableHeader('CVE');
                waitAndYieldRequestBodyVariables([getNodeVulnerabilitiesOpname]).then(
                    expectRequestedPagination({ offset: 0, limit: 20 })
                );
            }
        );
    });

    it('should update summary cards when a filter is applied', () => {
        interceptAndWatchRequests(routeMatcherMapForNodePage, staticResponseMapForNodePage).then(
            ({ waitForRequests }) => {
                visitFirstNodeFromOverviewPage();
                waitForRequests();

                applyLocalSeverityFilters('Low');
                cy.get(vulnSelectors.summaryCard('CVEs by severity')).contains('Critical hidden');
                cy.get(vulnSelectors.summaryCard('CVEs by severity')).contains('Important hidden');
                cy.get(vulnSelectors.summaryCard('CVEs by severity')).contains('Moderate hidden');
                cy.get(vulnSelectors.summaryCard('CVEs by severity')).contains(
                    new RegExp(/\d+ Low/)
                );
                filterHelpers.clearFilters();
                waitForRequests([getNodeVulnerabilitiesOpname]);

                applyLocalStatusFilters('Fixable');
                cy.get(vulnSelectors.summaryCard('CVEs by status')).contains(
                    new RegExp(/\d+ vulnerabilities with available fixes/)
                );
                cy.get(vulnSelectors.summaryCard('CVEs by status')).contains('Not fixable hidden');
            }
        );
    });
});
