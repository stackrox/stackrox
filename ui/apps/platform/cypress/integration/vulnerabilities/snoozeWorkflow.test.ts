import withAuth from '../../helpers/basicAuth';

// Note: each test case should be run against both Node and Platform CVE overview pages
describe('Node and Platform CVEs - Snooze workflow', () => {
    withAuth();

    it('should hide the snooze UI controls from users with NO_ACCESS to VulnerabilityManagementApprovals', () => {
        // check that users without VulnerabilityManagementApprovals access:
        //   - do not see table menu options
        //   - do not see bulk snooze actions
    });

    it('should hide the snooze UI controls from users with READ_ACCESS to VulnerabilityManagementApprovals', () => {
        // check that users with only read access to VulnerabilityManagementApprovals:
        //   - do not see table menu options
        //   - do not see bulk snooze actions
    });

    it('should show the snooze UI controls to users with WRITE_ACCESS to VulnerabilityManagementApprovals', () => {
        // check that users with write access to VulnerabilityManagementApprovals:
        //   - see table menu options
        //   - see bulk snooze actions
    });

    it('should hide the snooze UI controls for users with WRITE_ACCESS when the feature flag is disabled', () => {
        // disable the feature flag (NAME TBD) and check that users with write access to VulnerabilityManagementApprovals:
        //   - do not see table menu options
        //   - do not see bulk snooze actions
    });

    it('should allow users to snooze a single CVE', () => {
        // snooze a single CVE via the table menu
        // check that the CVE no longer exists in the table
        // click the "Show Snoozed" button and check that the snoozed CVE is displayed
        // click the "Hide Snoozed" button and check that the snoozed CVE is not displayed
        // click the "Show Snoozed" button again and unsnooze the CVE via the table menu
        // check that the CVE no longer exists in the table
        // click the "Hide Snoozed" button and check that the snoozed CVE is displayed
    });

    it('should allow users to snooze multiple CVEs', () => {
        // select multiple CVEs from the table
        // snooze multiple CVEs via the bulk snooze action
        // check that the CVEs no longer exist in the table
        // click the "Show Snoozed" button and check that the snoozed CVEs are displayed
        // click the "Hide Snoozed" button and check that the snoozed CVEs are not displayed
        // click the "Show Snoozed" button again and select the snoozed CVEs
        // unsnooze multiple CVEs via the bulk unsnooze action
        // check that the CVEs no longer exist in the table
        // click the "Hide Snoozed" button and check that the snoozed CVEs are displayed
    });

    it('should display the correct snooze duration in the table', () => {
        // for each of the following snooze durations [1 day, 1 week, 1 month, indefinite]:
        //   - perform the snooze action via the table menu
        //  - click the "Show Snoozed" button and check that the snoozed CVE is displayed
        //  - check that the table row contains a badge with the correct snooze duration
        //  - unsnooze the CVE via the table menu
    });
});
