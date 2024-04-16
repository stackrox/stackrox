import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';

describe('Platform CVEs - CVE Detail Page', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_VULN_MGMT_NODE_PLATFORM_CVES')) {
            this.skip();
        }
    });

    it('should restrict access to users with insufficient permissions', () => {
        // check that users without Cluster access cannot access the Platform CVE Detail page directly
    });

    it('should only show relevant filters for the Platform CVE Detail page', () => {
        // check the advanced filters and ensure only the relevant filters are displayed
    });

    it('should link to the correct pages', () => {
        // clicking the Platform CVEs breadcrumb should navigate to the overview page with the CVE tab selected
        // clicking a Cluster name in the list should navigate to the correct Cluster details page
    });

    it('should sort Cluster table columns', () => {
        // check sorting of Cluster column
        // check sorting of Cluster type column
        // check sorting of K8s version column
    });

    it('should filter the Cluster table', () => {
        // filtering by Cluster name should only display rows with a matching name
        // filtering by Cluster type should only display rows with a matching cluster type
        // filtering by K8s version should only display rows matching the version
        // clearing filters should remove all filter chips and filter from the URL
    });

    // Note: This might not be reliable in CI due to the low number of Clusters. We may need to mock and/or
    // just test the logic of the pagination in the URL.
    it('should correctly paginate the Cluster table', () => {
        // visit the location and save the list of Cluster names with a default perPage
        //   visit the location with perPage=2 in the URL
        //     should only display the first 2 rows of the previous list
        //     paginating to the next page should display the following two rows
        // go to page 1, then page 2
        //   applying a filter should reset the page to 1
        // go to page 1, then page 2
        //   go to next page, applying a sort should reset the page to 1
    });

    it('should correctly update summary cards when a filter is applied', () => {
        // apply a filter for a Cluster name and ensure that "affected clusters" contains the text: 1/n affected clusters
        // clear filters
        // apply a CVE status filter and ensure that the opposite status reads "Results hidden" in the card
    });
});
