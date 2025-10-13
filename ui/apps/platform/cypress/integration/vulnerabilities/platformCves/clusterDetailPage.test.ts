import withAuth from '../../../helpers/basicAuth';

describe('Platform CVEs - Cluster Detail Page', () => {
    withAuth();

    it('should restrict access to users with insufficient permissions', () => {
        // check that users without Cluster access cannot access the Cluster Detail page directly
    });

    it('should only show relevant filters for the Cluster Detail page', () => {
        // check the advanced filters and ensure only the relevant filters are displayed
    });

    it('should link to the correct pages', () => {
        // clicking the Cluster breadcrumb should navigate to the overview page with the Cluster tab selected
        // clicking a CVE name in the list should navigate to the correct Platform CVE details page
    });

    it('should sort CVE table columns', () => {
        // check sorting of CVE column
        // check sorting of CVE status column
        // check sorting of CVE type column
        // check sorting of CVSS column
    });

    it('should filter the CVE table', () => {
        // filtering by CVE name should only display rows with a matching name
        // filtering by CVE Status should only display rows with a matching status
        // filtering by CVE Type should only display rows with a matching type
        // filtering by CVSS should only display rows with a CVSS in range
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
        // apply a Fixable status filter and ensure that Not Fixable displays "Results hidden"
        // clear filters
        // apply a CVE Type filter and ensure that all other types read "Results hidden" in the card
    });
});
