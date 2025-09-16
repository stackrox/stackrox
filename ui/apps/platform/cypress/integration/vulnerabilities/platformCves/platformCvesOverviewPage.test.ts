import withAuth from '../../../helpers/basicAuth';

describe('Platform CVEs - Overview Page', () => {
    withAuth();

    it('should restrict access to users with insufficient permissions', () => {
        // check that users without Cluster access do not see Platform CVEs in the navigation
        // check that users without Cluster access cannot access the Platform CVEs page directly
    });

    it('should only show relevant filters for the Platform CVEs page', () => {
        // check the advanced filters and ensure only the relevant filters are displayed for CVEs
        // check the advanced filters and ensure only the relevant filters are displayed for Clusters
    });

    it('should link to the correct details pages', () => {
        // clicking a CVE in the list should navigate to the correct CVE details page
        // clicking a Cluster in the list should navigate to the correct Cluster details page
    });

    it('should sort CVE table columns', () => {
        // check sorting of CVE column
        // check sorting of CVE status column
        // check sorting of CVE type column
        // check sorting of CVSS column
        // check sorting of Affected Clusters column
    });

    it('should sort Cluster table columns', () => {
        // check sorting of Cluster column
        // check sorting of CVE count column
        // check sorting of Cluster type column
        // check sorting of K8s version column
    });

    it('should filter the CVE table', () => {
        // filtering by CVE name should only display rows with a matching name
        // filtering by Status should only display rows with a matching status
        // filtering by Type should only display rows with a matching type
        // filtering by CVSS should only display rows with a CVSS in range
        // clearing filters should remove all filter chips and filter from the URL
    });

    it('should filter the Cluster table', () => {
        // filtering by Cluster name should only display rows with a matching name
        // filtering by Cluster type should only display rows with a matching cluster type
        // filtering by K8s version should only display rows matching the version
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
        //   click the "clusters" tab, then click back to the "cves" tab should reset the page to 1
    });

    it('should correctly paginate the Cluster table', () => {
        // visit the location and save the list of Cluster names with a default perPage
        //   visit the location with perPage=2 in the URL
        //     should only display the first 2 rows of the previous list
        //     paginating to the next page should display the following two rows
        // go to page 1, then page 2
        //   applying a filter should reset the page to 1
        // go to page 1, then page 2
        //   go to next page, applying a sort should reset the page to 1
        // go to page 1, then page 2
        //   click the "cves" tab, then click back to the "clusters" tab should reset the page to 1
    });
});
