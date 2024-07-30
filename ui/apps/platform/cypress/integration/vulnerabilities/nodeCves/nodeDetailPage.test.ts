import withAuth from '../../../helpers/basicAuth';
import { assertAvailableFilters } from '../../../helpers/compoundFilters';
import { hasFeatureFlag } from '../../../helpers/features';
import {
    assertCannotFindThePage,
    visitWithStaticResponseForPermissions,
} from '../../../helpers/visit';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';
import {
    getNodeMetadataOpname,
    getNodeVulnerabilitiesOpname,
    getNodeVulnSummaryOpname,
    routeMatcherMapForNodePage,
    visitFirstNodeLinkFromTable,
    visitNodeCveOverviewPage,
} from './NodeCve.helpers';

const nodeBaseUrl = '/main/vulnerabilities/node-cves/nodes';
const mockNodeId = '1';
const mockNodeName = 'cypress-node-1';

const staticResponseMapForNodePage = {
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

    before(function () {
        if (!hasFeatureFlag('ROX_VULN_MGMT_NODE_PLATFORM_CVES')) {
            this.skip();
        }
    });

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
        visitNodeCveOverviewPage();
        cy.get(vulnSelectors.entityTypeToggleItem('Node')).click();

        visitFirstNodeLinkFromTable();

        assertAvailableFilters({
            CVE: ['Name', 'CVSS', 'Discovered Time'],
            'Node Component': ['Name', 'Version'],
        });
    });

    it('should link to the correct pages', () => {
        // clicking the Nodes breadcrumb should navigate to the overview page with the Node tab selected
        // clicking a CVE name in the list should navigate to the correct Node CVE details page
    });

    it('should sort CVE table columns', () => {
        // check sorting of CVE column
        // check sorting of Top Severity column
        // check sorting of CVE status column
        // check sorting of CVSS column
        // check sorting of Affected components column
        // check sorting of First discovered column
    });

    it('should filter the CVE table', () => {
        // filtering by CVE name should only display rows with a matching name
        // filtering by Severity should only display rows with a matching top severity
        // filtering by CVE Status should only display rows with a matching status
        // filtering by CVSS should only display rows with a CVSS in range
        // filtering by Component should only display rows with a nested table containing a matching component
        //   - expand each row
        //   - check that the component name exists in the table
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
    });

    it('should correctly update summary cards when a filter is applied', () => {
        // get the total number of nodes from the affected nodes card an store this value as 'n'
        // apply a filter for a Node name and ensure that "affected nodes" contains the text: 1/n affected nodes
        // clear filters
        // apply a Critical severity filter and ensure that Important/Moderate/Low severities read "Results hidden" in the card
    });
});
