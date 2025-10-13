import { graphql } from '../../../constants/apiEndpoints';
import withAuth from '../../../helpers/basicAuth';
import {
    cancelAllCveExceptions,
    fillAndSubmitExceptionForm,
} from '../workloadCves/WorkloadCves.helpers';
import { deferAndVisitRequestDetails } from './ExceptionManagement.helpers';

const deferralProps = {
    comment: 'Defer me',
    expiry: 'When any CVE is fixable',
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

    it('should be able to update a pending request', () => {
        deferAndVisitRequestDetails(deferralProps);

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
            .should('contain.text', deferralProps.comment);

        cy.wait('@getAffectedImagesCount');
        cy.wait('@getImageCVEList');

        // update deferral
        cy.get('button:contains("Update request")').click();
        cy.get('div[role="dialog"]').should('exist');
        fillAndSubmitExceptionForm(
            {
                comment: newComment,
                expiryLabel: newExpiry,
            },
            'PATCH'
        );

        cy.get('button:contains("Close")').click();
        cy.get('div[role="dialog"]').should('not.exist');
        cy.get('div.pf-v5-c-alert.pf-m-success').should(
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
