import withAuth from '../../../helpers/basicAuth';
import { assertAvailableFilters } from '../../../helpers/compoundFilters';
import { hasFeatureFlag } from '../../../helpers/features';
import {
    expectRequestedQuery,
    expectRequestedSort,
    getRouteMatcherMapForGraphQL,
    interactAndWaitForResponses,
    interceptAndWatchRequests,
} from '../../../helpers/request';
import * as filterHelpers from '../../../helpers/compoundFilters';
import {
    queryTableHeader,
    queryTableSortHeader,
    sortByTableHeader,
} from '../../../helpers/tableHelpers';
import { assertCannotFindThePage, visit } from '../../../helpers/visit';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';
import {
    getAffectedNodesOpname,
    getNodeCveMetadataOpname,
    getNodeCveSummaryOpname,
    nodeCveBaseUrl,
    routeMatcherMapForNodeCvePage,
    routeMatcherMapForNodeCves,
    routeMatcherMapForNodePage,
    visitNodeCvePage,
    visitNodeCvePageWithStaticPermissions,
} from './NodeCve.helpers';
import { staticResponseMapForNodePage } from './nodeDetailPage.test';
import { applyLocalSeverityFilters } from '../workloadCves/WorkloadCves.helpers';

const mockCveName = 'CYPRESS-CVE-2022-1996';

const staticResponseMapForNodeCvePage = {
    [getNodeCveMetadataOpname]: {
        fixture: `vulnerabilities/nodeCves/${getNodeCveMetadataOpname}`,
    },
    [getNodeCveSummaryOpname]: {
        fixture: `vulnerabilities/nodeCves/${getNodeCveSummaryOpname}`,
    },
    [getAffectedNodesOpname]: {
        fixture: `vulnerabilities/nodeCves/${getAffectedNodesOpname}`,
    },
};

describe('Node CVEs - CVE Detail Page', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_VULN_MGMT_NODE_PLATFORM_CVES')) {
            this.skip();
        }
    });

    it('should restrict access to users with insufficient "Node" permission', () => {
        visitNodeCvePageWithStaticPermissions(mockCveName, { Node: 'READ_ACCESS' });
        assertCannotFindThePage();
    });

    it('should restrict access to users with insufficient "Cluster" permission', () => {
        visitNodeCvePageWithStaticPermissions(mockCveName, { Cluster: 'READ_ACCESS' });
        assertCannotFindThePage();
    });

    it('should allow access to users with sufficient permissions', () => {
        visitNodeCvePageWithStaticPermissions(
            mockCveName,
            {
                Node: 'READ_ACCESS',
                Cluster: 'READ_ACCESS',
            },
            routeMatcherMapForNodeCvePage,
            staticResponseMapForNodeCvePage
        );
        cy.get('h1').contains(mockCveName);
    });

    it('should only show relevant filters for the page', () => {
        visit(
            `${nodeCveBaseUrl}/${mockCveName}`,
            getRouteMatcherMapForGraphQL(['getNodeCVEMetadata']),
            {}
        );
        assertAvailableFilters({
            Cluster: ['Name', 'Label', 'Type', 'Platform type'],
            Node: ['Name', 'Operating System', 'Label', 'Annotation', 'Scan Time'],
            'Node Component': ['Name', 'Version'],
        });
    });

    it('should link to the overview page from the breadcrumbs', () => {
        // clicking the Node CVEs breadcrumb should navigate to the overview page with the CVE tab selected
        visitNodeCvePage(
            mockCveName,
            routeMatcherMapForNodeCvePage,
            staticResponseMapForNodeCvePage
        );

        interactAndWaitForResponses(() => {
            cy.get('nav[aria-label="Breadcrumb"] a').contains('Node CVEs').click();
        }, routeMatcherMapForNodeCves);

        cy.get(`${vulnSelectors.entityTypeToggleItem('CVE')}[aria-pressed=true]`);
    });

    it('should link to the Node page from the name links in the table', () => {
        // clicking a Node name in the list should navigate to the correct Node details page
        visitNodeCvePage(
            mockCveName,
            routeMatcherMapForNodeCvePage,
            staticResponseMapForNodeCvePage
        );

        interactAndWaitForResponses(
            () => {
                cy.get(`table td[data-label="Node"]`).first().click();
            },
            routeMatcherMapForNodePage,
            staticResponseMapForNodePage
        );

        // Check for the presence of the Node breadcrumb link to ensure we are on the correct page
        cy.get('nav[aria-label="Breadcrumb"] a').contains('Nodes');
    });

    it('should sort Node table columns', () => {
        interceptAndWatchRequests(
            {
                [getAffectedNodesOpname]: routeMatcherMapForNodeCvePage[getAffectedNodesOpname],
            },
            {
                [getAffectedNodesOpname]: staticResponseMapForNodeCvePage[getAffectedNodesOpname],
            }
        ).then(({ waitForRequests, waitAndYieldRequestBodyVariables }) => {
            // Don't mock the metadata and summary requests, as they are not relevant to this test
            visitNodeCvePage(mockCveName);
            waitForRequests();

            // check sorting of Node column
            sortByTableHeader('Node');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedSort({ field: 'Node', reversed: true })
            );
            sortByTableHeader('Node');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedSort({ field: 'Node', reversed: false })
            );

            // check sorting of CVE Severity column
            sortByTableHeader('CVE severity');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedSort({ field: 'Severity', reversed: true })
            );
            sortByTableHeader('CVE severity');
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
            sortByTableHeader('CVSS score');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedSort({ field: 'CVSS', reversed: true })
            );
            sortByTableHeader('CVSS score');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedSort({ field: 'CVSS', reversed: false })
            );

            // check sorting of Cluster column
            sortByTableHeader('Cluster');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedSort({ field: 'Cluster', reversed: true })
            );
            sortByTableHeader('Cluster');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedSort({ field: 'Cluster', reversed: false })
            );

            // check sorting of Operating System column
            sortByTableHeader('Operating system');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedSort({ field: 'Operating System', reversed: true })
            );
            sortByTableHeader('Operating system');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedSort({ field: 'Operating System', reversed: false })
            );

            // check that the Affected components column is not sortable
            queryTableHeader('Affected components');
            queryTableSortHeader('Affected components').should('not.exist');
        });
    });

    it('should filter the Node table', () => {
        interceptAndWatchRequests(
            {
                [getAffectedNodesOpname]: routeMatcherMapForNodeCvePage[getAffectedNodesOpname],
            },
            {
                [getAffectedNodesOpname]: staticResponseMapForNodeCvePage[getAffectedNodesOpname],
            }
        ).then(({ waitForRequests, waitAndYieldRequestBodyVariables }) => {
            // Don't mock the metadata and summary requests, as they are not relevant to this test
            visitNodeCvePage(mockCveName);
            waitForRequests();

            // filtering by Node name should only display rows with a matching name
            filterHelpers.addAutocompleteFilter('Node', 'Name', 'cypress-node-1');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedQuery(`Node:r/cypress-node-1+CVE:r/^${mockCveName}$`)
            );
            filterHelpers.clearFilters();
            waitForRequests();

            applyLocalSeverityFilters('Low');
            waitAndYieldRequestBodyVariables().then(
                expectRequestedQuery(`SEVERITY:LOW_VULNERABILITY_SEVERITY+CVE:r/^${mockCveName}$`)
            );
            cy.get(vulnSelectors.summaryCard('Nodes by severity')).contains('Critical hidden');
            cy.get(vulnSelectors.summaryCard('Nodes by severity')).contains('Important hidden');
            cy.get(vulnSelectors.summaryCard('Nodes by severity')).contains('Moderate hidden');
            cy.get(vulnSelectors.summaryCard('Nodes by severity')).contains(new RegExp(/\d+ Low/));
            filterHelpers.clearFilters();
            waitForRequests();
        });
    });

    // Note: This might not be reliable in CI due to the low number of Nodes. We may need to mock and/or
    // just test the logic of the pagination in the URL.
    it('should paginate the Node table', () => {
        // visit the location and save the list of Node names with a default perPage
        //   visit the location with perPage=2 in the URL
        //     should only display the first 2 rows of the previous list
        //     paginating to the next page should display the following two rows
        // go to page 1, then page 2
        //   applying a filter should reset the page to 1
        // go to page 1, then page 2
        //   go to next page, applying a sort should reset the page to 1
    });

    it('should update summary cards when a filter is applied', () => {
        // apply a Critical severity filter and ensure that Important/Moderate/Low severities read "Results hidden" in the card
        // clear filters
        // apply a CVE status filter and ensure that the opposite status reads "Results hidden" in the card
    });

    it('should allow viewing the Node details', () => {
        // click the Details tab and ensure that the vulnerabilities table no longer displays
        // verify that a Cluster field exists
        // verify that label and annotation sections exist
    });
});
