import withAuth from '../../../helpers/basicAuth';
import { cancelAllCveExceptions } from '../workloadCves/WorkloadCves.helpers';
import { deferAndVisitRequestDetails, approveRequest } from './ExceptionManagement.helpers';

const deferralProps = {
    comment: 'Defer me',
    expiry: 'When all CVEs are fixable',
    scope: 'All images',
};

describe('Exception Management Request Details Page', () => {
    withAuth();

    beforeEach(() => {
        cancelAllCveExceptions();
    });

    after(() => {
        cancelAllCveExceptions();
    });

    it('should be able to approve a request if approval permissions are granted', () => {
        deferAndVisitRequestDetails(deferralProps);
        approveRequest();
    });
});
