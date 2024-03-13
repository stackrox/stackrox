import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import { cancelAllCveExceptions } from '../workloadCves/WorkloadCves.helpers';
import {
    deferAndVisitRequestDetails,
    markFalsePositiveAndVisitRequestDetails,
    visitPendingRequestsTab,
    approveRequest,
} from './ExceptionManagement.helpers';
import { selectors } from './ExceptionManagement.selectors';

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

    it('should be able to cancel a request if the user is the requester', () => {
        deferAndVisitRequestDetails({
            comment,
            expiry,
            scope,
        });
        cy.get('button:contains("Cancel request")').click();
        cy.get('div[role="dialog"]').should('exist');
        cy.get('div[role="dialog"] button:contains("Cancel request")').click();
        cy.get('div[role="dialog"]').should('not.exist');
        cy.get('div[aria-label="Success Alert"]').should(
            'contain',
            'The vulnerability request was successfully canceled.'
        );
        cy.get('div[aria-label="Warning Alert"]').should(
            'contain',
            'You are viewing a canceled request. If this cancelation was not intended, please submit a new request'
        );
    });

    it('should be able to see how many CVEs will be affected by a cancel', () => {
        deferAndVisitRequestDetails({
            comment,
            expiry,
            scope,
        });
        cy.get('table tbody tr:not(".pf-c-table__expandable-row")').then((rows) => {
            const numCVEs = rows.length;
            cy.get('button:contains("Cancel request")').click();
            cy.get('div[role="dialog"]').should('exist');
            cy.get(`div:contains("CVE count: ${numCVEs}")`).should('exist');
        });
    });

    it('should not see a cancelled request in the approved deferrals table', () => {
        deferAndVisitRequestDetails({
            comment,
            expiry,
            scope,
        });
        approveRequest();
        cy.get('button:contains("Cancel request")').click();
        cy.get('div[role="dialog"]').should('exist');
        cy.get('div[role="dialog"] button:contains("Cancel request")').click();
        cy.get('div[role="dialog"]').should('not.exist');
        visitPendingRequestsTab();
        cy.get(selectors.approvedDeferralsTab).click();
        cy.get(
            'table tbody div.pf-c-empty-state__content h2:contains("No approved deferral requests")'
        ).should('exist');
        cy.get(
            'table tbody div.pf-c-empty-state__content div.pf-c-empty-state__body p:contains("There are currently no approved deferral requests. Feel free to review pending requests or return to your dashboard.")'
        ).should('exist');
    });

    it('should not see a cancelled request in the approved false positives table', () => {
        markFalsePositiveAndVisitRequestDetails({
            comment,
            scope,
        });
        approveRequest();
        cy.get('button:contains("Cancel request")').click();
        cy.get('div[role="dialog"]').should('exist');
        cy.get('div[role="dialog"] button:contains("Cancel request")').click();
        cy.get('div[role="dialog"]').should('not.exist');
        visitPendingRequestsTab();
        cy.get(selectors.approvedFalsePositivesTab).click();
        cy.get(
            'table tbody div.pf-c-empty-state__content h2:contains("No approved false positive requests")'
        ).should('exist');
        cy.get(
            'table tbody div.pf-c-empty-state__content div.pf-c-empty-state__body p:contains("There are currently no approved false positive requests. Feel free to review pending requests or return to your dashboard.")'
        ).should('exist');
    });
});
