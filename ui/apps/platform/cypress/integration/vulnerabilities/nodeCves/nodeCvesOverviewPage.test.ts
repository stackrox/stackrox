import withAuth from '../../../helpers/basicAuth';
import { assertAvailableFilters } from '../../../helpers/compoundFilters';
import {
    assertCannotFindThePage,
    visitWithStaticResponseForPermissions,
} from '../../../helpers/visit';
import navSelectors from '../../../selectors/navigation';
import {
    getEntityTypeCountsOpname,
    getNodeCvesOpname,
    getNodesOpname,
    routeMatcherMapForEntityCounts,
    routeMatcherMapForNodeCves,
    routeMatcherMapForNodes,
    visitFirstNodeLinkFromTable,
    visitNodeCveOverviewPage,
} from './NodeCve.helpers';
import {
    assertOnEachRowForColumn,
    paginateNext,
    paginatePrevious,
    queryTableHeader,
    queryTableSortHeader,
    sortByTableHeader,
} from '../../../helpers/tableHelpers';
import {
    expectRequestedPagination,
    expectRequestedQuery,
    expectRequestedSort,
    interceptAndWatchRequests,
} from '../../../helpers/request';
import * as filterHelpers from '../../../helpers/compoundFilters';
import { applyLocalSeverityFilters } from '../workloadCves/WorkloadCves.helpers';
import { getSeverityLabelCounts, visitEntityTab } from '../vulnerabilities.helpers';

describe('Node CVEs - Overview Page', () => {
    withAuth();

    it('should restrict access to users with insufficient "Cluster" permission', () => {
        // When lacking the minimum permissions:
        // - Check that the Node CVEs link is not visible in the left navigation
        // - Check that direct navigation fails

        // Missing 'Cluster' permission
        visitWithStaticResponseForPermissions('/main/vulnerabilities/user-workloads', {
            body: { resourceToAccess: { Node: 'READ_ACCESS' } },
        });
        // With this limited permission, the entire horizontal nav should be missing
        cy.get(navSelectors.horizontalNavBar).should('not.exist');
        visitNodeCveOverviewPage();
        assertCannotFindThePage();
    });

    it('should restrict access to users with insufficient "Node" permission', () => {
        // When lacking the minimum permissions:
        // - Check that the Node CVEs link is not visible in the left navigation
        // - Check that direct navigation fails
        // Missing 'Node' permission
        visitWithStaticResponseForPermissions('/main/vulnerabilities/user-workloads', {
            body: { resourceToAccess: { Cluster: 'READ_ACCESS' } },
        });
        cy.get(`${navSelectors.horizontalNavLinks}:contains('Nodes')`).should('not.exist');
        visitNodeCveOverviewPage();
        assertCannotFindThePage();
    });

    it('should allow access to users with sufficient "Node" and "Cluster" permissions', () => {
        // Has both 'Node' and 'Cluster' permissions
        visitWithStaticResponseForPermissions('/main/vulnerabilities/user-workloads', {
            body: { resourceToAccess: { Node: 'READ_ACCESS', Cluster: 'READ_ACCESS' } },
        });
        // Link should be visible in the left navigation
        // Clicking the link should navigate to the Node CVEs page
        cy.get(navSelectors.horizontalNavLinks).contains('Nodes').click();
        cy.get('h1').contains('Node CVEs');
    });

    it('should only show relevant filters for the Node CVEs page', () => {
        visitNodeCveOverviewPage();
        const expectedFilters = {
            CVE: ['Name', 'CVSS', 'Discovered Time'],
            Node: ['Name', 'Operating System', 'Label', 'Annotation', 'Scan Time'],
            'Node Component': ['Name', 'Version'],
            Cluster: ['ID', 'Name', 'Label', 'Type', 'Platform type'],
        };

        // check the advanced filters and ensure only the relevant filters are displayed for CVEs
        assertAvailableFilters(expectedFilters);

        // check the advanced filters and ensure only the relevant filters are displayed for Nodes
        visitEntityTab('Node');
        assertAvailableFilters(expectedFilters);
    });

    it('should link a CVE table row to the correct CVE detail page', () => {
        // Having a CVE in CI is unreliable, so we mock the request and assert
        // on the link construction instead of the content of the detail page.
        visitNodeCveOverviewPage(routeMatcherMapForNodeCves, {
            [getNodeCvesOpname]: {
                fixture: `vulnerabilities/nodeCves/${getNodeCvesOpname}`,
            },
        });

        cy.get('tbody tr td[data-label="CVE"] a')
            .first()
            .then(($link) => {
                const linkHref = $link.attr('href');
                const linkName = $link.text();
                expect(linkHref).to.match(new RegExp(`.*/${linkName}$`));
            });
    });

    it('should link a Node table row to the correct Node detail page', () => {
        visitNodeCveOverviewPage();
        visitEntityTab('Node');

        visitFirstNodeLinkFromTable().then((name) => {
            cy.get('h1').contains(name);
        });
    });

    it('should sort CVE table columns', () => {
        // check sorting of CVE column
        interceptAndWatchRequests(routeMatcherMapForNodeCves).then(
            ({ waitForRequests, waitAndYieldRequestBodyVariables }) => {
                visitNodeCveOverviewPage();
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

                // check that the Nodes by severity column is not sortable
                queryTableHeader('Nodes by severity');
                queryTableSortHeader('Nodes by severity').should('not.exist');

                // check sorting of Top CVSS column
                sortByTableHeader('Top CVSS');
                waitAndYieldRequestBodyVariables().then(
                    expectRequestedSort({
                        field: 'CVSS',
                        reversed: true,
                        aggregateBy: { aggregateFunc: 'max', distinct: false },
                    })
                );

                sortByTableHeader('Top CVSS');
                waitAndYieldRequestBodyVariables().then(
                    expectRequestedSort({
                        field: 'CVSS',
                        reversed: false,
                        aggregateBy: { aggregateFunc: 'max', distinct: false },
                    })
                );

                // check sorting of Affected nodes column
                sortByTableHeader('Affected nodes');
                waitAndYieldRequestBodyVariables().then(
                    expectRequestedSort({
                        field: 'Node ID',
                        reversed: true,
                        aggregateBy: { aggregateFunc: 'count', distinct: true },
                    })
                );

                sortByTableHeader('Affected nodes');
                waitAndYieldRequestBodyVariables().then(
                    expectRequestedSort({
                        field: 'Node ID',
                        reversed: false,
                        aggregateBy: { aggregateFunc: 'count', distinct: true },
                    })
                );

                // check that the First discovered column is not sortable
                queryTableHeader('First discovered');
                queryTableSortHeader('First discovered').should('not.exist');
            }
        );
    });

    it('should sort Node table columns', () => {
        interceptAndWatchRequests(routeMatcherMapForNodes).then(
            ({
                waitForRequests,
                waitAndYieldRequestBodyVariables: waitAndInspectRequestVariables,
            }) => {
                // Visit Node tab and wait for initial load - sorting will be pre-applied to the Node column
                visitNodeCveOverviewPage();
                visitEntityTab('Node');
                waitForRequests();

                // check sorting of Node column
                sortByTableHeader('Node');
                waitAndInspectRequestVariables().then(
                    expectRequestedSort({ field: 'Node', reversed: true })
                );

                sortByTableHeader('Node');
                waitAndInspectRequestVariables().then(
                    expectRequestedSort({ field: 'Node', reversed: false })
                );

                // check that CVEs by Severity is not sortable
                queryTableHeader('CVEs by severity');
                queryTableSortHeader('CVEs by severity').should('not.exist');

                // check sorting of Cluster column
                sortByTableHeader('Cluster');
                waitAndInspectRequestVariables().then(
                    expectRequestedSort({ field: 'Cluster', reversed: true })
                );

                sortByTableHeader('Cluster');
                waitAndInspectRequestVariables().then(
                    expectRequestedSort({ field: 'Cluster', reversed: false })
                );

                // check sorting of Operating System column
                sortByTableHeader('Operating system');
                waitAndInspectRequestVariables().then(
                    expectRequestedSort({ field: 'Operating System', reversed: true })
                );

                sortByTableHeader('Operating system');
                waitAndInspectRequestVariables().then(
                    expectRequestedSort({ field: 'Operating System', reversed: false })
                );

                // check sorting of Scan time column
                sortByTableHeader('Scan time');
                waitAndInspectRequestVariables().then(
                    expectRequestedSort({ field: 'Node Scan Time', reversed: true })
                );

                sortByTableHeader('Scan time');
                waitAndInspectRequestVariables().then(
                    expectRequestedSort({ field: 'Node Scan Time', reversed: false })
                );
            }
        );
    });

    it('should filter the CVE table', () => {
        interceptAndWatchRequests(routeMatcherMapForNodeCves).then(
            ({ waitForRequests, waitAndYieldRequestBodyVariables }) => {
                // Visit Node tab and wait for initial load
                visitNodeCveOverviewPage();
                waitForRequests();

                // filtering by CVE name should only display rows with a matching name
                filterHelpers.addAutocompleteFilter('CVE', 'Name', 'CVE-2021-1234');
                waitAndYieldRequestBodyVariables().then(
                    expectRequestedQuery('CVE:r/CVE-2021-1234')
                );
                // Do not assert on cell contents as the filter value is mocked
                filterHelpers.clearFilters();
                waitForRequests();

                // filtering by Severity should only display rows with a matching severity
                // filtering by Severity should not report counts for hidden severities
                applyLocalSeverityFilters('Low');
                waitAndYieldRequestBodyVariables().then(
                    expectRequestedQuery('SEVERITY:LOW_VULNERABILITY_SEVERITY')
                );
                assertOnEachRowForColumn('Nodes by severity', (_, cell) => {
                    const { critical, important, moderate, low } = getSeverityLabelCounts(cell);
                    expect(critical).to.be.null;
                    expect(important).to.be.null;
                    expect(moderate).to.be.null;
                    expect(low).to.be.greaterThan(0);
                });
                filterHelpers.clearFilters();
                waitForRequests();

                // filtering by CVSS should only display rows with a CVSS in range
                filterHelpers.addNumericFilter('CVE', 'CVSS', 'Is less than', 8);
                waitAndYieldRequestBodyVariables().then(expectRequestedQuery('CVSS:<8'));
                assertOnEachRowForColumn('Top CVSS', (_, cell) => {
                    const cvss = parseFloat(cell.innerText.replace(/[^0-9.]/g, ''));
                    expect(cvss).to.be.lessThan(8);
                });
                filterHelpers.clearFilters();
                waitForRequests();

                // filtering by CVE Discovered Time should only display rows matching the timeframe
                // TODO - Implement once we support date ranges, otherwise this is of little utility

                // applying multiple filters should combine queries in the request
                filterHelpers.addAutocompleteFilter('CVE', 'Name', 'CVE-2021-1234');
                waitForRequests();
                filterHelpers.addNumericFilter('CVE', 'CVSS', 'Is less than', 8);
                waitAndYieldRequestBodyVariables().then(
                    expectRequestedQuery('CVE:r/CVE-2021-1234+CVSS:<8')
                );
            }
        );
    });

    it('should filter the Node table', () => {
        interceptAndWatchRequests(routeMatcherMapForNodes).then(
            ({ waitForRequests, waitAndYieldRequestBodyVariables }) => {
                // Visit Node tab and wait for initial load
                visitNodeCveOverviewPage();
                visitEntityTab('Node');
                waitForRequests();

                // filtering by Node name should only display rows with a matching name
                const nodeNameFilter = 'a';
                filterHelpers.addAutocompleteFilter('Node', 'Name', nodeNameFilter);
                waitAndYieldRequestBodyVariables().then(
                    expectRequestedQuery(`Node:r/${nodeNameFilter}`)
                );
                assertOnEachRowForColumn('Node', (_, cell) => {
                    expect(cell.innerText).to.match(new RegExp(nodeNameFilter, 'i'));
                });
                filterHelpers.clearFilters();
                waitForRequests();

                // filtering by Severity should only display rows with a matching severity
                // filtering by Severity should not report counts for hidden severities
                applyLocalSeverityFilters('Low');
                waitAndYieldRequestBodyVariables().then(
                    expectRequestedQuery('SEVERITY:LOW_VULNERABILITY_SEVERITY')
                );
                assertOnEachRowForColumn('CVEs by severity', (_, cell) => {
                    const { critical, important, moderate, low } = getSeverityLabelCounts(cell);
                    expect(critical).to.be.null;
                    expect(important).to.be.null;
                    expect(moderate).to.be.null;
                    expect(low).to.be.greaterThan(0);
                });
                filterHelpers.clearFilters();
                waitForRequests();

                // filtering by Cluster should only display rows with a matching cluster
                const clusterNameFilter = 'a';
                filterHelpers.addAutocompleteFilter('Cluster', 'Name', clusterNameFilter);
                waitAndYieldRequestBodyVariables().then(
                    expectRequestedQuery(`Cluster:r/${clusterNameFilter}`)
                );
                assertOnEachRowForColumn('Cluster', (_, cell) => {
                    expect(cell.innerText).to.match(new RegExp(clusterNameFilter, 'i'));
                });
                filterHelpers.clearFilters();
                waitForRequests();

                // filtering by Operating System should only display rows with a matching OS
                const osFilter = 'red hat';
                filterHelpers.addPlainTextFilter('Node', 'Operating System', osFilter);
                waitAndYieldRequestBodyVariables().then(
                    expectRequestedQuery(`Operating System:r/${osFilter}`)
                );
                assertOnEachRowForColumn('Operating system', (_, cell) => {
                    expect(cell.innerText).to.match(new RegExp(osFilter, 'i'));
                });
                filterHelpers.clearFilters();
                waitForRequests();

                // filtering by Scan Time should only display rows matching the timeframe
                // TODO - Implement once we support date ranges, otherwise this is of little utility

                // applying multiple filters should combine queries in the request
                filterHelpers.addAutocompleteFilter('Node', 'Name', nodeNameFilter);
                waitForRequests();
                filterHelpers.addAutocompleteFilter('Cluster', 'Name', clusterNameFilter);
                waitAndYieldRequestBodyVariables().then(
                    expectRequestedQuery(`Node:r/${nodeNameFilter}+Cluster:r/${clusterNameFilter}`)
                );
            }
        );
    });

    it('should correctly paginate the CVE table', () => {
        interceptAndWatchRequests(
            {
                ...routeMatcherMapForEntityCounts,
                ...routeMatcherMapForNodes,
                ...routeMatcherMapForNodeCves,
            },
            {
                [getEntityTypeCountsOpname]: {
                    body: { data: { nodeCVECount: 100, nodeCount: 100 } },
                },
            }
        ).then(({ waitForRequests, waitAndYieldRequestBodyVariables }) => {
            visitNodeCveOverviewPage();
            waitForRequests([getNodeCvesOpname]);

            // test pagination basics
            paginateNext();
            waitAndYieldRequestBodyVariables([getNodeCvesOpname]).then(
                expectRequestedPagination({ offset: 20, limit: 20 })
            );

            paginateNext();
            waitAndYieldRequestBodyVariables([getNodeCvesOpname]).then(
                expectRequestedPagination({ offset: 40, limit: 20 })
            );

            paginatePrevious();
            waitAndYieldRequestBodyVariables([getNodeCvesOpname]).then(
                expectRequestedPagination({ offset: 20, limit: 20 })
            );

            paginatePrevious();
            waitAndYieldRequestBodyVariables([getNodeCvesOpname]).then(
                expectRequestedPagination({ offset: 0, limit: 20 })
            );

            // test that applying a filter resets the page to 1
            paginateNext();
            waitForRequests([getNodeCvesOpname]);
            filterHelpers.addAutocompleteFilter('CVE', 'Name', 'CVE-2021-1234');
            waitAndYieldRequestBodyVariables([getNodeCvesOpname, getEntityTypeCountsOpname]).then(
                ([nodeCveListVars]) => {
                    expectRequestedPagination({ offset: 0, limit: 20 })(nodeCveListVars);
                }
            );

            // test that applying a sort resets the page to 1
            paginateNext();
            waitForRequests([getNodeCvesOpname]);
            sortByTableHeader('CVE');
            waitAndYieldRequestBodyVariables([getNodeCvesOpname]).then(
                expectRequestedPagination({ offset: 0, limit: 20 })
            );

            // test that pagination is reset when switching between tabs
            paginateNext();
            waitForRequests([getNodeCvesOpname]);
            visitEntityTab('Node');
            waitAndYieldRequestBodyVariables([getNodesOpname]).then(
                expectRequestedPagination({ offset: 0, limit: 20 })
            );
            visitEntityTab('CVE');
            waitAndYieldRequestBodyVariables([getNodeCvesOpname]).then(
                expectRequestedPagination({ offset: 0, limit: 20 })
            );
        });
    });

    it('should correctly paginate the Node table', () => {
        interceptAndWatchRequests(
            {
                ...routeMatcherMapForEntityCounts,
                ...routeMatcherMapForNodes,
                ...routeMatcherMapForNodeCves,
            },
            {
                [getEntityTypeCountsOpname]: {
                    body: { data: { nodeCVECount: 100, nodeCount: 100 } },
                },
            }
        ).then(({ waitForRequests, waitAndYieldRequestBodyVariables }) => {
            visitNodeCveOverviewPage();
            visitEntityTab('Node');
            waitForRequests([getNodesOpname]);

            // test pagination basics
            paginateNext();
            waitAndYieldRequestBodyVariables([getNodesOpname]).then(
                expectRequestedPagination({ offset: 20, limit: 20 })
            );

            paginateNext();
            waitAndYieldRequestBodyVariables([getNodesOpname]).then(
                expectRequestedPagination({ offset: 40, limit: 20 })
            );

            paginatePrevious();
            waitAndYieldRequestBodyVariables([getNodesOpname]).then(
                expectRequestedPagination({ offset: 20, limit: 20 })
            );

            paginatePrevious();
            waitAndYieldRequestBodyVariables([getNodesOpname]).then(
                expectRequestedPagination({ offset: 0, limit: 20 })
            );

            // test that applying a filter resets the page to 1
            paginateNext();
            waitForRequests([getNodesOpname]);
            filterHelpers.addAutocompleteFilter('Node', 'Name', 'a');
            waitAndYieldRequestBodyVariables([getNodesOpname, getEntityTypeCountsOpname]).then(
                ([nodeListVars]) => {
                    expectRequestedPagination({ offset: 0, limit: 20 })(nodeListVars);
                }
            );

            // test that applying a sort resets the page to 1
            paginateNext();
            waitForRequests([getNodesOpname]);
            sortByTableHeader('Node');
            waitAndYieldRequestBodyVariables([getNodesOpname]).then(
                expectRequestedPagination({ offset: 0, limit: 20 })
            );

            // test that pagination is reset when switching between tabs
            paginateNext();
            waitForRequests([getNodesOpname]);
            visitEntityTab('CVE');
            waitAndYieldRequestBodyVariables([getNodeCvesOpname]).then(
                expectRequestedPagination({ offset: 0, limit: 20 })
            );
            visitEntityTab('Node');
            waitAndYieldRequestBodyVariables([getNodesOpname]).then(
                expectRequestedPagination({ offset: 0, limit: 20 })
            );
        });
    });
});
