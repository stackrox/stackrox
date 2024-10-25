import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import { cancelAllCveExceptions } from '../workloadCves/WorkloadCves.helpers';
import { deferAndVisitRequestDetails, approveRequest } from './ExceptionManagement.helpers';

const deferralProps = {
    comment: 'Defer me',
    expiry: 'When all CVEs are fixable',
    scope: 'All images',
};

describe('Exception Management Request Details Page', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES')) {
            this.skip();
        }
    });

    beforeEach(() => {
        if (hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES')) {
            cancelAllCveExceptions();
        }
    });

    after(() => {
        if (hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES')) {
            cancelAllCveExceptions();
        }
    });

    it('should be able to approve a request if approval permissions are granted', () => {
        deferAndVisitRequestDetails(deferralProps);
        approveRequest();
    });
});
