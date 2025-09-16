import withAuth from '../../../helpers/basicAuth';

// TODO - classic tests conditionally skip many of these checks when running on OpenShift, we
//        need to verify if this is still necessary and if so, how to handle it in the new tests
describe('Platform CVEs - CVE Detail Page', () => {
    withAuth();

    it('should restrict access to users with insufficient permissions', () => {
        // check that users without Cluster access cannot access the Platform CVE Detail page directly
    });

    it('should only show relevant filters for the page', () => {
        // check the advanced filters and ensure only the relevant filters are displayed
    });

    it('should link to the overview page from the breadcrumbs', () => {
        // clicking the Platform CVEs breadcrumb should navigate to the overview page with the CVE tab selected
    });

    it('should link to the cluster page from the name links in the table', () => {
        // clicking a Cluster name in the list should navigate to the correct Cluster details page
    });

    it('should display the expected Cluster table columns', () => {
        // check presence of Cluster column
        // check presence of Cluster type column
        // check presence of K8s version column
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
    it('should paginate the Cluster table', () => {
        // visit the location and save the list of Cluster names with a default perPage
        //   visit the location with perPage=2 in the URL
        //     should only display the first 2 rows of the previous list
        //     paginating to the next page should display the following two rows
        // go to page 1, then page 2
        //   applying a filter should reset the page to 1
        // go to page 1, then page 2
        //   go to next page, applying a sort should reset the page to 1
    });

    it('should update summary cards when a filter is applied', () => {
        // apply a filter for a Cluster name and ensure that "affected clusters" contains the text: 1/n affected clusters
        // clear filters
        // apply a CVE status filter and ensure that the opposite status reads "Results hidden" in the card
    });
});
