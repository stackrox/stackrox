import withAuth from '../../../helpers/basicAuth';
import { cancelAllCveExceptions } from '../workloadCves/WorkloadCves.helpers';
import { deferAndVisitRequestDetails, denyRequest } from './ExceptionManagement.helpers';

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

    it('should be able to deny a request if approval permissions are granted', () => {
        deferAndVisitRequestDetails(deferralProps);
        denyRequest();
        // should not be able to cancel a denied request
        cy.get('button:contains("Cancel request")').should('not.exist');
    });

    it('should be able to see how many CVEs will be affected by a denial', () => {
        deferAndVisitRequestDetails(deferralProps);
        cy.get('table tbody tr:not(".pf-v5-c-table__expandable-row")').then((rows) => {
            const numCVEs = rows.length;
            cy.get('button:contains("Deny request")').click();
            cy.get('div[role="dialog"]').should('exist');
            cy.get(`div:contains("CVE count: ${numCVEs}")`).should('exist');
        });
    });
});
