import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import {
    cancelAllCveExceptions,
    typeAndSelectCustomSearchFilterValue,
} from '../workloadCves/WorkloadCves.helpers';
import {
    deferAndVisitRequestDetails,
    visitExceptionManagement,
} from './ExceptionManagement.helpers';
import { approveRequest } from './approveRequestFlow.test';
import { selectors } from './ExceptionManagement.selectors';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';

const comment = 'Defer me';
const expiry = 'When all CVEs are fixable';
const scope = 'All images';

function deferAndApprove() {
    deferAndVisitRequestDetails({
        comment,
        expiry,
        scope,
    });
    approveRequest();
}

function visitApprovedDeferralsTab() {
    visitExceptionManagement();
    cy.get(selectors.approvedDeferralsTab).click();
    // Wait for the loading spinner to disappear
    cy.get('.pf-c-spinner').should('not.exist');
}

describe('Exception Management Pending Requests Page', () => {
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

    it('should be able to view approved deferrals', () => {
        deferAndApprove();
        visitApprovedDeferralsTab();

        // the deferred request should be approved
        cy.get(
            'table td[data-label="Requested action"]:contains("Deferred (when all fixed)")'
        ).should('exist');
    });

    it('should be able to navigate to the Request Details page by clicking on the request name', () => {
        deferAndApprove();
        visitApprovedDeferralsTab();

        const requestNameSelector = 'table td[data-label="Request name"]';

        cy.get(requestNameSelector)
            .invoke('text')
            .then((requestName) => {
                cy.get(requestNameSelector).click();
                cy.get(`h1:contains("${requestName}")`).should('exist');
            });
    });

    it('should be able to sort on the "Request Name" column', () => {
        visitApprovedDeferralsTab();

        cy.get(selectors.tableSortColumn('Request name')).should(
            'have.attr',
            'aria-sort',
            'descending'
        );
        cy.get(selectors.tableColumnSortButton('Request name')).click();
        cy.location('search').should(
            'contain',
            'sortOption[field]=Request%20Name&sortOption[direction]=asc'
        );
        cy.get(selectors.tableSortColumn('Request name')).should(
            'have.attr',
            'aria-sort',
            'ascending'
        );
        cy.get(selectors.tableColumnSortButton('Request name')).click();
        cy.location('search').should(
            'contain',
            'sortOption[field]=Request%20Name&sortOption[direction]=desc'
        );
        cy.get(selectors.tableSortColumn('Request name')).should(
            'have.attr',
            'aria-sort',
            'descending'
        );
    });

    it('should be able to sort on the "Requester" column', () => {
        visitApprovedDeferralsTab();

        cy.get(selectors.tableSortColumn('Requester')).should('have.attr', 'aria-sort', 'none');
        cy.get(selectors.tableColumnSortButton('Requester')).click();
        cy.location('search').should(
            'contain',
            'sortOption[field]=Requester%20User%20Name&sortOption[direction]=desc'
        );
        cy.get(selectors.tableSortColumn('Requester')).should(
            'have.attr',
            'aria-sort',
            'descending'
        );
        cy.get(selectors.tableColumnSortButton('Requester')).click();
        cy.location('search').should(
            'contain',
            'sortOption[field]=Requester%20User%20Name&sortOption[direction]=asc'
        );
        cy.get(selectors.tableSortColumn('Requester')).should(
            'have.attr',
            'aria-sort',
            'ascending'
        );
    });

    it('should be able to sort on the "Requested" column', () => {
        visitApprovedDeferralsTab();

        cy.get(selectors.tableSortColumn('Requested')).should('have.attr', 'aria-sort', 'none');
        cy.get(selectors.tableColumnSortButton('Requested')).click();
        cy.location('search').should(
            'contain',
            'sortOption[field]=Created%20Time&sortOption[direction]=desc'
        );
        cy.get(selectors.tableSortColumn('Requested')).should(
            'have.attr',
            'aria-sort',
            'descending'
        );
        cy.get(selectors.tableColumnSortButton('Requested')).click();
        cy.location('search').should(
            'contain',
            'sortOption[field]=Created%20Time&sortOption[direction]=asc'
        );
        cy.get(selectors.tableSortColumn('Requested')).should(
            'have.attr',
            'aria-sort',
            'ascending'
        );
    });

    it('should be able to sort on the "Expires" column', () => {
        visitApprovedDeferralsTab();

        cy.get(selectors.tableSortColumn('Expires')).should('have.attr', 'aria-sort', 'none');
        cy.get(selectors.tableColumnSortButton('Expires')).click();
        cy.location('search').should(
            'contain',
            'sortOption[field]=Request%20Expiry%20Time&sortOption[direction]=desc'
        );
        cy.get(selectors.tableSortColumn('Expires')).should('have.attr', 'aria-sort', 'descending');
        cy.get(selectors.tableColumnSortButton('Expires')).click();
        cy.location('search').should(
            'contain',
            'sortOption[field]=Request%20Expiry%20Time&sortOption[direction]=asc'
        );
        cy.get(selectors.tableSortColumn('Expires')).should('have.attr', 'aria-sort', 'ascending');
    });

    it('should be able to sort on the "Scope" column', () => {
        visitApprovedDeferralsTab();

        cy.get(selectors.tableSortColumn('Scope')).should('have.attr', 'aria-sort', 'none');
        cy.get(selectors.tableColumnSortButton('Scope')).click();
        cy.location('search').should(
            'contain',
            'sortOption[field]=Image%20Registry%20Scope&sortOption[direction]=desc'
        );
        cy.get(selectors.tableSortColumn('Scope')).should('have.attr', 'aria-sort', 'descending');
        cy.get(selectors.tableColumnSortButton('Scope')).click();
        cy.location('search').should(
            'contain',
            'sortOption[field]=Image%20Registry%20Scope&sortOption[direction]=asc'
        );
        cy.get(selectors.tableSortColumn('Scope')).should('have.attr', 'aria-sort', 'ascending');
    });

    it('should be able to filter by "Request name"', () => {
        deferAndApprove();
        visitApprovedDeferralsTab();

        cy.get('table td[data-label="Request name"] a').then((element) => {
            const requestName = element.text().trim();
            typeAndSelectCustomSearchFilterValue('Request name', requestName);
            cy.get('table td[data-label="Request name"] a').should('exist');
        });
    });

    it('should be able to filter by "Request name"', () => {
        deferAndApprove();
        visitApprovedDeferralsTab();

        typeAndSelectCustomSearchFilterValue('Requester', 'ui_tests');
        cy.get('table td[data-label="Request name"] a').should('exist');
        cy.get(vulnSelectors.clearFiltersButton).click();
        typeAndSelectCustomSearchFilterValue('Requester', 'BLAH');
        cy.get('table td[data-label="Request name"] a').should('not.exist');
    });
});
