import withAuth from '../../../helpers/basicAuth';
import { assertAvailableFilters } from '../../../helpers/compoundFilters';
import { hasFeatureFlag } from '../../../helpers/features';
import {
    assertCannotFindThePage,
    visitWithStaticResponseForPermissions,
} from '../../../helpers/visit';
import navSelectors from '../../../selectors/navigation';
import { visitNodeCveOverviewPage } from './NodeCve.helpers';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';

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

    it('should link to the correct details pages', () => {
        // clicking a CVE in the list should navigate to the correct CVE details page
        // clicking a Node in the list should navigate to the correct Node details page
    });

    it('should sort CVE table columns', () => {
        // check sorting of CVE column
        // check sorting of Nodes by Severity column
        // check sorting of Top CVSS column
        // check sorting of Affected Nodes column
        // check sorting of First discovered column
    });

    it('should sort Node table columns', () => {
        // check sorting of Node column
        // check sorting of CVEs by Severity column
        // check sorting of Cluster column
        // check sorting of Operating System column
        // check sorting of Scan time column
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
