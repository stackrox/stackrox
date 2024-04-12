import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import { cancelAllCveExceptions } from '../workloadCves/WorkloadCves.helpers';
import { deferAndVisitRequestDetails, denyRequest } from './ExceptionManagement.helpers';

const comment = 'Defer me';
const expiry = 'When all CVEs are fixable';
const scope = 'All images';

describe('Exception Management Request Details Page', () => {
    withAuth();

    before(function () {
        if (
            !hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES') ||
            !hasFeatureFlag('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL')
        ) {
            this.skip();
        }
    });

    beforeEach(() => {
        if (
            hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES') &&
            hasFeatureFlag('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL')
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
            hasFeatureFlag('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL')
        ) {
            cancelAllCveExceptions();
        }
    });

    it('should be able to deny a request if approval permissions are granted', () => {
        denyRequest();
        // should not be able to cancel a denied request
        cy.get('button:contains("Cancel request")').should('not.exist');
    });

    it('should be able to see how many CVEs will be affected by a denial', () => {
        cy.get('table tbody tr:not(".pf-v5-c-table__expandable-row")').then((rows) => {
            const numCVEs = rows.length;
            cy.get('button:contains("Deny request")').click();
            cy.get('div[role="dialog"]').should('exist');
            cy.get(`div:contains("CVE count: ${numCVEs}")`).should('exist');
        });
    });
});
