import { graphql } from '../../../constants/apiEndpoints';
import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import {
    cancelAllCveExceptions,
    fillAndSubmitExceptionForm,
} from '../workloadCves/WorkloadCves.helpers';
import { deferAndVisitRequestDetails } from './ExceptionManagement.helpers';

const comment = 'Defer me';
const expiry = 'When any CVE is fixable';
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

    it('should be able to update a pending request', () => {
        const newExpiry = 'When all CVEs are fixable';
        const newComment = 'Updated';

        cy.intercept({ method: 'POST', url: graphql('getAffectedImagesCount') }).as(
            'getAffectedImagesCount'
        );
        cy.intercept({ method: 'POST', url: graphql('getImageCVEList') }).as('getImageCVEList');

        // check values pre-update
        cy.get('dl.vulnerability-exception-request-overview')
            .contains('dt', 'Requested action')
            .next('dd')
            .should('have.text', 'Deferred (when any fixed)');
        cy.get('dl.vulnerability-exception-request-overview')
            .contains('dt', 'Latest comment')
            .next('dd')
            .should('contain.text', comment);

        cy.wait('@getAffectedImagesCount');
        cy.wait('@getImageCVEList');

        // update deferral
        cy.get('button:contains("Update request")').click();
        cy.get('div[role="dialog"]').should('exist');
        fillAndSubmitExceptionForm({
            comment: newComment,
            expiryLabel: newExpiry,
        });

        cy.get('button:contains("Close")').click();
        cy.get('div[role="dialog"]').should('not.exist');
        cy.get('div[aria-label="Success Alert"]').should(
            'contain',
            'The vulnerability request was successfully updated.'
        );

        // check values post-update
        cy.get('dl.vulnerability-exception-request-overview')
            .contains('dt', 'Requested action')
            .next('dd')
            .should('have.text', 'Deferred (when all fixed)');
        cy.get('dl.vulnerability-exception-request-overview')
            .contains('dt', 'Latest comment')
            .next('dd')
            .should('contain.text', newComment);
    });
});
