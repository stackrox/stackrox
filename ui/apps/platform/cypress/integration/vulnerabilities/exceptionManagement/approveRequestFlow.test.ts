import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import { getInputByLabel } from '../../../helpers/formHelpers';
import { cancelAllCveExceptions } from '../workloadCves/WorkloadCves.helpers';
import { deferAndVisitRequestDetails } from './ExceptionManagement.helpers';

const comment = 'Defer me';
const expiry = 'When all CVEs are fixable';
const scope = 'All images';

describe('Exception Management Request Details Page', () => {
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
            deferAndVisitRequestDetails({
                comment,
                expiry,
                scope,
            });
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

    it('should be able to approve a request if approval permissions are granted', () => {
        cy.get('button:contains("Approve request")').click();
        cy.get('div[role="dialog"]').should('exist');
        getInputByLabel('Approval rationale').type('Approved');
        cy.get('div[role="dialog"] button:contains("Approve")').click();
        cy.get('div[role="dialog"]').should('not.exist');
        cy.get('div[aria-label="Success Alert"]').should(
            'contain',
            'The vulnerability request was successfully approved.'
        );
    });
});
