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
import { selectors as workloadCVEsSelectors } from '../workloadCves/WorkloadCves.selectors';
import { visitExceptionManagement } from './ExceptionManagement.helpers';

function clearWorkloadCVEsFilters() {
    cy.get('body').then((body) => {
        if (body.find(workloadCVEsSelectors.clearFiltersButton).length > 0) {
            // If button exists, click it
            cy.get(workloadCVEsSelectors.clearFiltersButton).click(); // Note: This is a workaround to prevent a lack of CVE data from causing the test to fail in CI
        }
    });
}

describe('Exception Management Pending Requests Page', () => {
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

    it('should be able to view deferred pending requests', () => {
        visitWorkloadCveOverview();
        clearWorkloadCVEsFilters();

        // defer a single cve
        selectSingleCveForException('DEFERRAL').then((cveName) => {
            verifySelectedCvesInModal([cveName]);
            fillAndSubmitExceptionForm({
                comment: 'Test comment',
                expiryLabel: 'When all CVEs are fixable',
            });
            verifyExceptionConfirmationDetails({
                expectedAction: 'Deferral',
                cves: [cveName],
                scope: 'All images',
                expiry: 'When all CVEs are fixable',
            });

            visitExceptionManagement();

            // the deferred request should be pending
            cy.get(
                'table td[data-label="Requested action"]:contains("Deferred (when all fixed)")'
            ).should('exist');
        });
    });

    it('should be able to view false positive pending requests', () => {
        visitWorkloadCveOverview();
        clearWorkloadCVEsFilters();

        // mark a single cve as false positive
        selectSingleCveForException('FALSE_POSITIVE').then((cveName) => {
            verifySelectedCvesInModal([cveName]);
            fillAndSubmitExceptionForm({ comment: 'Test comment' });
            verifyExceptionConfirmationDetails({
                expectedAction: 'False positive',
                cves: [cveName],
                scope: 'All images',
            });

            visitExceptionManagement();

            // the false positive request should be pending
            cy.get('table td[data-label="Requested action"]:contains("False positive")').should(
                'exist'
            );
        });
    });

    it('should be able to navigate to the Request Details page by clicking on the request name', () => {
        visitWorkloadCveOverview();
        clearWorkloadCVEsFilters();

        selectSingleCveForException('FALSE_POSITIVE')
            // mark a single cve as false positive
            .then((cveName) => {
                verifySelectedCvesInModal([cveName]);
                fillAndSubmitExceptionForm({ comment: 'Test comment' });
                verifyExceptionConfirmationDetails({
                    expectedAction: 'False positive',
                    cves: [cveName],
                    scope: 'All images',
                });
            })
            .then(() => {
                visitExceptionManagement();

                const requestNameLink = 'table td[data-label="Request name"]';

                cy.get(requestNameLink)
                    .invoke('text')
                    .then((requestName) => {
                        cy.get(requestNameLink).click();
                        cy.get(`h1:contains("${requestName}")`).should('exist');
                    });
            });
    });
});
