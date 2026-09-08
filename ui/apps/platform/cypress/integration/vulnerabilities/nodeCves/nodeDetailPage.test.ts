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
    getNodeVulnerabilitiesOpname,
    routeMatcherMapForNodePage,
    routeMatcherMapForNodes,
    staticResponseMapForNodePage,
    visitNodePage,
} from './NodeCve.helpers';

const { assertAvailableFilters } = filterHelpers;

const nodeBaseUrl = '/main/vulnerabilities/node-cves/nodes';
const mockNodeId = '1';
const mockNodeName = 'cypress-node-1';

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
        visitNodePage(mockNodeId, routeMatcherMapForNodePage, staticResponseMapForNodePage);

        assertAvailableFilters({
            CVE: ['Name', 'CVSS', 'Discovered Time'],
            'Node Component': ['Name', 'Version'],
        });
    });

    it('should follow the breadcrumb link to the Node list tab', () => {
        visitNodePage(mockNodeId, routeMatcherMapForNodePage, staticResponseMapForNodePage);

        // clicking the Nodes breadcrumb should navigate to the overview page with the Node tab selected
        interactAndWaitForResponses(() => {
            cy.get('nav[aria-label="Breadcrumb"] a').contains('Nodes').click();
        }, routeMatcherMapForNodes);
        cy.get(`${vulnSelectors.entityTypeToggleItem('CVE')}[aria-pressed=false]`);
        cy.get(`${vulnSelectors.entityTypeToggleItem('Node')}[aria-pressed=true]`);
    });

    it('should link from a CVE in the table to the CVE detail page', () => {
        visitNodePage(mockNodeId, routeMatcherMapForNodePage, staticResponseMapForNodePage);

        // clicking a CVE name in the list should navigate to a Node CVE details page
        cy.get(`table td[data-label="CVE"]`).first().click();
        cy.get('nav[aria-label="Breadcrumb"] a').contains('Node CVEs');
    });

    it('should sort CVE table columns', () => {
        interceptAndWatchRequests(routeMatcherMapForNodePage, staticResponseMapForNodePage).then(
            ({ waitForRequests, waitAndYieldRequestBodyVariables }) => {
                visitNodePage(mockNodeId);
                waitForRequests();

                const waitForVulnQuery = () =>
                    waitAndYieldRequestBodyVariables([getNodeVulnerabilitiesOpname]);

                // check sorting of CVE column
                sortByTableHeader('CVE');
                waitForVulnQuery().then(expectRequestedSort({ field: 'CVE', reversed: true }));
                sortByTableHeader('CVE');
                waitForVulnQuery().then(expectRequestedSort({ field: 'CVE', reversed: false }));

                // check sorting of Top Severity column
                sortByTableHeader('Top CVE severity');
                waitForVulnQuery().then(expectRequestedSort({ field: 'Severity', reversed: true }));
                sortByTableHeader('Top CVE severity');
                waitForVulnQuery().then(
                    expectRequestedSort({ field: 'Severity', reversed: false })
                );

                // check sorting of CVE status column
                sortByTableHeader('CVE status');
                waitForVulnQuery().then(expectRequestedSort({ field: 'Fixable', reversed: true }));
                sortByTableHeader('CVE status');
                waitForVulnQuery().then(expectRequestedSort({ field: 'Fixable', reversed: false }));

                // check sorting of CVSS column
                sortByTableHeader('Top CVSS');
                waitForVulnQuery().then(expectRequestedSort({ field: 'CVSS', reversed: true }));
                sortByTableHeader('Top CVSS');
                waitForVulnQuery().then(expectRequestedSort({ field: 'CVSS', reversed: false }));

                // check that the Affected components column is not sortable
                queryTableHeader('Affected components');
                queryTableSortHeader('Affected components').should('not.exist');
            }
        );
    });

    it('should filter the CVE table', () => {
        interceptAndWatchRequests(routeMatcherMapForNodePage, staticResponseMapForNodePage).then(
            ({ waitForRequests, waitAndYieldRequestBodyVariables }) => {
                visitNodePage(mockNodeId);
                waitForRequests();

                const waitForVulnQuery = () =>
                    waitAndYieldRequestBodyVariables([getNodeVulnerabilitiesOpname]);

                // Assert GraphQL query strings. Stubbed rows are not a live filter result.
                filterHelpers.addAutocompleteFilter('CVE', 'Name', 'CVE-2021-1234');
                waitForVulnQuery().then(expectRequestedQuery('CVE:r/CVE-2021-1234'));
                filterHelpers.clearFilters();
                waitForRequests([getNodeVulnerabilitiesOpname]);

                applyLocalSeverityFilters('Low');
                waitForVulnQuery().then(
                    expectRequestedQuery('Severity:LOW_VULNERABILITY_SEVERITY')
                );
                filterHelpers.clearFilters();
                waitForRequests([getNodeVulnerabilitiesOpname]);

                applyLocalStatusFilters('Fixable');
                waitForVulnQuery().then(expectRequestedQuery('Fixable:true'));
                filterHelpers.clearFilters();
                waitForRequests([getNodeVulnerabilitiesOpname]);

                filterHelpers.addNumericFilter('CVE', 'CVSS', 'Is less than', 8);
                waitForVulnQuery().then(expectRequestedQuery('CVSS:<8'));
                filterHelpers.clearFilters();
                waitForRequests([getNodeVulnerabilitiesOpname]);

                const componentFilter = 'a';
                filterHelpers.addAutocompleteFilter('Node component', 'Name', componentFilter);
                waitForVulnQuery().then(expectRequestedQuery(`Component:r/${componentFilter}`));
            }
        );
    });

    it('should correctly paginate the CVE table', () => {
        interceptAndWatchRequests(routeMatcherMapForNodePage, staticResponseMapForNodePage).then(
            ({ waitForRequests, waitAndYieldRequestBodyVariables }) => {
                visitNodePage(mockNodeId);
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
                visitNodePage(mockNodeId);
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
