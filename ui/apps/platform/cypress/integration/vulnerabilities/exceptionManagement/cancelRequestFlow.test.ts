import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import {
    cancelAllCveExceptions,
    fillAndSubmitExceptionForm,
    selectSingleCveForException,
    verifyExceptionConfirmationDetails,
    verifySelectedCvesInModal,
    visitWorkloadCveOverview,
} from '../workloadCves/WorkloadCves.helpers';
import { selectors as workloadCVESelectors } from '../workloadCves/WorkloadCves.selectors';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';
import { visit } from '../../../helpers/visit';
import { pendingRequestsPath } from './ExceptionManagement.helpers';

const deferralComment = 'Defer me';
const deferralExpiry = 'When all CVEs are fixable';
const deferralScope = 'All images';

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

            visitWorkloadCveOverview();
            cy.get(vulnSelectors.clearFiltersButton).click(); // Note: This is a workaround to prevent a lack of CVE data from causing the test to fail in CI

            // defer a single cve
            selectSingleCveForException('DEFERRAL').then((cveName) => {
                verifySelectedCvesInModal([cveName]);
                fillAndSubmitExceptionForm({
                    comment: deferralComment,
                    expiryLabel: deferralExpiry,
                });
                verifyExceptionConfirmationDetails({
                    expectedAction: 'Deferral',
                    cves: [cveName],
                    scope: deferralScope,
                    expiry: deferralExpiry,
                });
                cy.get(workloadCVESelectors.copyToClipboardButton).click();
                cy.get(workloadCVESelectors.copyToClipboardTooltipText).contains('Copied');
                // @TODO: Can make this into a custom cypress command (ie. getClipboardText)
                cy.window()
                    .then((win) => {
                        return win.navigator.clipboard.readText();
                    })
                    .then((url) => {
                        visit(url);
                    });
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

    it('should be able to cancel a request if the user is the requester', () => {
        cy.get('button:contains("Cancel request")').click();
        cy.get('div[role="dialog"]').should('exist');
        cy.get('div[role="dialog"] button:contains("Cancel request")').click();
        cy.get('div[role="dialog"]').should('not.exist');
        cy.location().should((location) => {
            expect(location.pathname).to.eq(pendingRequestsPath);
        });
    });

    it('should be able to see how many CVEs will be affected by a cancel', () => {
        cy.get('table tbody tr:not(".pf-c-table__expandable-row")').then((rows) => {
            const numCVEs = rows.length;
            cy.get('button:contains("Cancel request")').click();
            cy.get('div[role="dialog"]').should('exist');
            cy.get(`div:contains("CVE count: ${numCVEs}")`).should('exist');
        });
    });
});
