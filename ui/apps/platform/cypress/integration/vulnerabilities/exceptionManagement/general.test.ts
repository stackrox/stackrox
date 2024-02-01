import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import { getRegExpForTitleWithBranding } from '../../../helpers/title';
import { visit } from '../../../helpers/visit';
import { cancelAllCveExceptions } from '../workloadCves/WorkloadCves.helpers';
import {
    approvedDeferralsPath,
    approvedFalsePositivesPath,
    deferAndVisitRequestDetails,
    deniedRequestsPath,
    pendingRequestsPath,
} from './ExceptionManagement.helpers';

describe('Exception Management', () => {
    withAuth();

    before(function () {
        if (
            !hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES') ||
            !hasFeatureFlag('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL') ||
            !hasFeatureFlag('ROX_WORKLOAD_CVES_FIXABILITY_FILTERS')
        ) {
            this.skip();
        }
    });

    beforeEach(() => {
        if (
            hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES') &&
            hasFeatureFlag('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL') &&
            hasFeatureFlag('ROX_WORKLOAD_CVES_FIXABILITY_FILTERS')
        ) {
            cancelAllCveExceptions();
        }
    });

    after(() => {
        if (
            hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES') &&
            hasFeatureFlag('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL') &&
            hasFeatureFlag('ROX_WORKLOAD_CVES_FIXABILITY_FILTERS')
        ) {
            cancelAllCveExceptions();
        }
    });

    it('should have the correct browser title for pending requests', () => {
        visit(pendingRequestsPath);
        cy.title().should(
            'match',
            getRegExpForTitleWithBranding('Exception Management - Pending Requests')
        );
    });

    it('should have the correct browser title for approved deferrals', () => {
        visit(approvedDeferralsPath);
        cy.title().should(
            'match',
            getRegExpForTitleWithBranding('Exception Management - Approved Deferrals')
        );
    });

    it('should have the correct browser title for approved false positives', () => {
        visit(approvedFalsePositivesPath);
        cy.title().should(
            'match',
            getRegExpForTitleWithBranding('Exception Management - Approved False Positives')
        );
    });

    it('should have the correct browser title for denied requests', () => {
        visit(deniedRequestsPath);
        cy.title().should(
            'match',
            getRegExpForTitleWithBranding('Exception Management - Denied Requests')
        );
    });

    it('should have the correct browser title for request details', () => {
        const comment = 'Defer me';
        const expiry = 'When all CVEs are fixable';
        const scope = 'All images';
        deferAndVisitRequestDetails({
            comment,
            expiry,
            scope,
        });
        cy.title().should(
            'match',
            getRegExpForTitleWithBranding('Exception Management - Request Details')
        );
    });
});
