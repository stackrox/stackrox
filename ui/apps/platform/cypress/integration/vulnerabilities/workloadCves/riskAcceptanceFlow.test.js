import withAuth from '../../../helpers/basicAuth';

// TODO - These tests are intended to test the filtering/linking/etc of the Workload CVE
// pages as it pertains to 'Vulnerability State' (Observed/Deferred/False Positive). Once the VM 2.0
// Risk Acceptance workload is added, these tests should be filled out to ensure all flows are working correctly
// and that the correct CVEs are being displayed on the Workload CVE pages.
describe.skip('Workload CVE Risk Acceptance flow', () => {
    withAuth();

    describe('Observed/Deferred/False Positive CVEs', () => {
        it('should correctly filter Observed CVEs on Workload CVE pages', () => {});
        it('should correctly filter Deferred CVEs on Workload CVE pages', () => {});
        it('should correctly filter False Positive CVEs on Workload CVE pages', () => {});
    });
});
