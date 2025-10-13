import { graphql } from '../../../constants/apiEndpoints';
import withAuth from '../../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../../helpers/title';
import { visit } from '../../../helpers/visit';
import {
    cancelAllCveExceptions,
    typeAndEnterCustomSearchFilterValue,
} from '../workloadCves/WorkloadCves.helpers';
import {
    approvedDeferralsPath,
    approvedFalsePositivesPath,
    deferAndVisitRequestDetails,
    deniedRequestsPath,
    pendingRequestsPath,
    visitPendingRequestsTab,
} from './ExceptionManagement.helpers';
import { selectors } from './ExceptionManagement.selectors';

describe('Exception Management', () => {
    withAuth();

    beforeEach(() => {
        cancelAllCveExceptions();
    });

    after(() => {
        cancelAllCveExceptions();
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

    it('should keep filters when navigating between tabs', () => {
        const filterText = 'AA-240101-1';

        cy.intercept({ method: 'POST', url: graphql('autocomplete') }).as('autocomplete');

        visitPendingRequestsTab();

        // Add a filter
        typeAndEnterCustomSearchFilterValue('Exception', 'Request Name', filterText);

        // The filter should be applied
        cy.get('div[aria-label="applied search filters"]').should('exist');

        // switch to Approved deferrals tab
        cy.get(selectors.approvedDeferralsTab).click();

        // The filter should be applied
        cy.get('div[aria-label="applied search filters"]').should('exist');

        // switch to Approved false positives tab
        cy.get(selectors.approvedFalsePositivesTab).click();

        // The filter should be applied
        cy.get('div[aria-label="applied search filters"]').should('exist');

        // switch to Denied requests tab
        cy.get(selectors.deniedRequestsTab).click();

        // The filter should be applied
        cy.get('div[aria-label="applied search filters"]').should('exist');
    });
});
