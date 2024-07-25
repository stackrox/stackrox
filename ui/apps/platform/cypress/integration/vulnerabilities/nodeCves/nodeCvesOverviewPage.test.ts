import withAuth from '../../../helpers/basicAuth';
import { assertAvailableFilters } from '../../../helpers/compoundFilters';
import { hasFeatureFlag } from '../../../helpers/features';
import {
    assertCannotFindThePage,
    visitWithStaticResponseForPermissions,
} from '../../../helpers/visit';
import navSelectors from '../../../selectors/navigation';
import {
    mockOverviewNodeCveListRequest,
    visitFirstNodeLinkFromTable,
    visitNodeCveOverviewPage,
} from './NodeCve.helpers';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';
import {
    queryTableHeader,
    queryTableSortHeader,
    sortByTableHeader,
} from '../../../helpers/tableHelpers';
import { expectRequestedSort } from '../../../helpers/sort';
import { waitForTableLoadCompleteIndicator } from '../workloadCves/WorkloadCves.helpers';
import { interactAndInspectGraphQLVariables } from '../../../helpers/request';

describe('Node CVEs - Overview Page', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_VULN_MGMT_NODE_PLATFORM_CVES')) {
            this.skip();
        }
    });

    it('should restrict access to users with insufficient permissions', () => {
        // When lacking the minimum permissions:
        // - Check that the Node CVEs link is not visible in the left navigation
        // - Check that direct navigation fails

        // Missing 'Cluster' permission
        visitWithStaticResponseForPermissions('/main', {
            body: { resourceToAccess: { Node: 'READ_ACCESS' } },
        });
        cy.get(navSelectors.allNavLinks).contains('Node CVEs').should('not.exist');
        visitNodeCveOverviewPage();
        assertCannotFindThePage();

        // Missing 'Node' permission
        visitWithStaticResponseForPermissions('/main', {
            body: { resourceToAccess: { Cluster: 'READ_ACCESS' } },
        });
        cy.get(navSelectors.allNavLinks).contains('Node CVEs').should('not.exist');
        visitNodeCveOverviewPage();
        assertCannotFindThePage();

        // Has both 'Node' and 'Cluster' permissions
        visitWithStaticResponseForPermissions('/main', {
            body: { resourceToAccess: { Node: 'READ_ACCESS', Cluster: 'READ_ACCESS' } },
        });
        // Link should be visible in the left navigation
        cy.get(navSelectors.allNavLinks).contains('Node CVEs');
        // Clicking the link should navigate to the Node CVEs page
        cy.get(navSelectors.navExpandableVulnerabilityManagement).click();
        cy.get(navSelectors.nestedNavLinks).contains('Node CVEs').click();
        cy.get('h1').contains('Node CVEs');
    });

    it('should only show relevant filters for the Node CVEs page', () => {
        visitNodeCveOverviewPage();
        const expectedFilters = {
            CVE: ['Name', 'CVSS', 'Discovered Time'],
            Node: ['Name', 'Operating System', 'Label', 'Annotation', 'Scan Time'],
            'Node Component': ['Name', 'Version'],
            Cluster: ['Name', 'Label', 'Type', 'Platform type'],
        };

        // check the advanced filters and ensure only the relevant filters are displayed for CVEs
        assertAvailableFilters(expectedFilters);

        // check the advanced filters and ensure only the relevant filters are displayed for Nodes
        cy.get(vulnSelectors.entityTypeToggleItem('Node')).click();
        assertAvailableFilters(expectedFilters);
    });

    it('should link a CVE table row to the correct CVE detail page', () => {
        // Having a CVE in CI is unreliable, so we mock the request and assert
        // on the link construction instead of the content of the detail page.
        mockOverviewNodeCveListRequest();
        visitNodeCveOverviewPage();

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
        cy.get(vulnSelectors.entityTypeToggleItem('Node')).click();

        visitFirstNodeLinkFromTable().then((name) => {
            cy.get('h1').contains(name);
        });
    });

    it('should sort CVE table columns', () => {
        visitNodeCveOverviewPage();
        waitForTableLoadCompleteIndicator();

        // check sorting of CVE column
        interactAndInspectGraphQLVariables(() => sortByTableHeader('CVE'), 'getNodeCVEs').then(
            expectRequestedSort({ field: 'CVE', reversed: true })
        );
        interactAndInspectGraphQLVariables(() => sortByTableHeader('CVE'), 'getNodeCVEs').then(
            expectRequestedSort({ field: 'CVE', reversed: false })
        );

        // check that the Nodes by severity column is not sortable
        queryTableHeader('Nodes by severity');
        queryTableSortHeader('Nodes by severity').should('not.exist');

        // check sorting of Top CVSS column
        interactAndInspectGraphQLVariables(() => sortByTableHeader('Top CVSS'), 'getNodeCVEs').then(
            expectRequestedSort({
                field: 'CVSS',
                reversed: true,
                aggregateBy: { aggregateFunc: 'max', distinct: false },
            })
        );
        interactAndInspectGraphQLVariables(() => sortByTableHeader('Top CVSS'), 'getNodeCVEs').then(
            expectRequestedSort({
                field: 'CVSS',
                reversed: false,
                aggregateBy: { aggregateFunc: 'max', distinct: false },
            })
        );

        // check sorting of Affected Nodes column
        interactAndInspectGraphQLVariables(
            () => sortByTableHeader('Affected nodes'),
            'getNodeCVEs'
        ).then(
            expectRequestedSort({
                field: 'Node ID',
                reversed: true,
                aggregateBy: { aggregateFunc: 'count', distinct: true },
            })
        );
        interactAndInspectGraphQLVariables(
            () => sortByTableHeader('Affected nodes'),
            'getNodeCVEs'
        ).then(
            expectRequestedSort({
                field: 'Node ID',
                reversed: false,
                aggregateBy: { aggregateFunc: 'count', distinct: true },
            })
        );

        // check that the First discovered column is not sortable
        queryTableHeader('First discovered');
        queryTableSortHeader('First discovered').should('not.exist');
    });

    it('should sort Node table columns', () => {
        // Visit Node tab and wait for initial load - sorting will be pre-applied to the Node column
        interactAndInspectGraphQLVariables(() => {
            visitNodeCveOverviewPage();
            cy.get(vulnSelectors.entityTypeToggleItem('Node')).click();
        }, 'getNodes');

        // check sorting of Node column
        interactAndInspectGraphQLVariables(() => sortByTableHeader('Node'), 'getNodes').then(
            expectRequestedSort({ field: 'Node', reversed: true })
        );
        interactAndInspectGraphQLVariables(() => sortByTableHeader('Node'), 'getNodes').then(
            expectRequestedSort({ field: 'Node', reversed: false })
        );

        // check that CVEs by Severity is not sortable
        queryTableHeader('CVEs by severity');
        queryTableSortHeader('CVEs by severity').should('not.exist');

        // check sorting of Cluster column
        interactAndInspectGraphQLVariables(() => sortByTableHeader('Cluster'), 'getNodes').then(
            expectRequestedSort({ field: 'Cluster', reversed: true })
        );
        interactAndInspectGraphQLVariables(() => sortByTableHeader('Cluster'), 'getNodes').then(
            expectRequestedSort({ field: 'Cluster', reversed: false })
        );

        // check sorting of Operating System column
        interactAndInspectGraphQLVariables(
            () => sortByTableHeader('Operating system'),
            'getNodes'
        ).then(expectRequestedSort({ field: 'Operating System', reversed: true }));
        interactAndInspectGraphQLVariables(
            () => sortByTableHeader('Operating system'),
            'getNodes'
        ).then(expectRequestedSort({ field: 'Operating System', reversed: false }));

        // check sorting of Scan time column
        interactAndInspectGraphQLVariables(() => sortByTableHeader('Scan time'), 'getNodes').then(
            expectRequestedSort({ field: 'Node Scan Time', reversed: true })
        );
        interactAndInspectGraphQLVariables(() => sortByTableHeader('Scan time'), 'getNodes').then(
            expectRequestedSort({ field: 'Node Scan Time', reversed: false })
        );
    });

    it('should filter the CVE table', () => {
        // filtering by CVE name should only display rows with a matching name
        // filtering by Severity should only display rows with a matching severity
        // filtering by Severity should change icons to "hidden severity" icons
        // filtering by CVSS should only display rows with a CVSS in range
        // filtering by CVE Discovered Time should only display rows matching the timeframe
        // clearing filters should remove all filter chips and filter from the URL
    });

    it('should filter the Node table', () => {
        // filtering by Node name should only display rows with a matching name
        // filtering by Severity should only display rows with a matching severity
        // filtering by Severity should change icons to "hidden severity" icons
        // filtering by Cluster should only display rows with a matching cluster
        // filtering by Operating System should only display rows with a matching OS
        // filtering by Scan Time should only display rows matching the timeframe
        // clearing filters should remove all filter chips and filter from the URL
    });

    it('should correctly paginate the CVE table', () => {
        // visit the location and save the list of CVE names with a default perPage
        //   visit the location with perPage=2 in the URL
        //     should only display the first 2 rows of the previous list
        //     paginating to the next page should display the following two rows
        // go to page 1, then page 2
        //   applying a filter should reset the page to 1
        // go to page 1, then page 2
        //   go to next page, applying a sort should reset the page to 1
        // go to page 1, then page 2
        //   click the "nodes" tab, then click back to the "cves" tab should reset the page to 1
    });

    it('should correctly paginate the Node table', () => {
        // visit the location and save the list of Node names with a default perPage
        //   visit the location with perPage=2 in the URL
        //     should only display the first 2 rows of the previous list
        //     paginating to the next page should display the following two rows
        // go to page 1, then page 2
        //   applying a filter should reset the page to 1
        // go to page 1, then page 2
        //   go to next page, applying a sort should reset the page to 1
        // go to page 1, then page 2
        //   click the "cves" tab, then click back to the "nodes" tab should reset the page to 1
    });
});
