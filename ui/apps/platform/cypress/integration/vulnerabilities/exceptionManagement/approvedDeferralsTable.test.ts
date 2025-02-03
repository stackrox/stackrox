import withAuth from '../../../helpers/basicAuth';
import {
    cancelAllCveExceptions,
    typeAndEnterCustomSearchFilterValue,
    viewCvesByObservationState,
    visitWorkloadCveOverview,
} from '../workloadCves/WorkloadCves.helpers';
import {
    deferAndVisitRequestDetails,
    visitApprovedDeferralsTab,
    approveRequest,
} from './ExceptionManagement.helpers';
import { selectors } from './ExceptionManagement.selectors';
import { selectors as workloadSelectors } from '../workloadCves/WorkloadCves.selectors';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';

const comment = 'Defer me';
const expiry = 'When all CVEs are fixable';
const scope = 'All images';

describe('Exception Management - Approved Deferrals Table', () => {
    withAuth();

    beforeEach(() => {
        cancelAllCveExceptions();
    });

    after(() => {
        cancelAllCveExceptions();
    });

    it('should be able to view approved deferrals', () => {
        deferAndVisitRequestDetails({
            comment,
            expiry,
            scope,
        });
        approveRequest();
        visitApprovedDeferralsTab();

        // the deferred request should be approved
        cy.get(
            'table tr:nth(1) td[data-label="Requested action"]:contains("Deferred (when all fixed)")'
        ).should('exist');
    });

    it('should navigate from Workload CVEs to a request list filtered by the specific CVE', () => {
        deferAndVisitRequestDetails({ comment, expiry, scope }).then(({ requestName, cveName }) => {
            approveRequest();

            visitWorkloadCveOverview();
            viewCvesByObservationState('Deferred');

            // Verify correct CVE filter
            cy.get('td[data-label="Request details"] a:contains("View")').click();
            cy.get(workloadSelectors.filterChipGroupItem('CVE', cveName));

            // Verify a link in the table containing the request
            cy.get('td a').contains(requestName);
        });
    });

    it('should be able to navigate to the Request Details page by clicking on the request name', () => {
        deferAndVisitRequestDetails({
            comment,
            expiry,
            scope,
        });
        approveRequest();
        visitApprovedDeferralsTab();

        const requestNameSelector = 'table tr:nth(1) td[data-label="Request name"]';

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

    // TODO: We can create one test for all sorting. Consider making a reusable function for all the other table tests
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
        deferAndVisitRequestDetails({
            comment,
            expiry,
            scope,
        });
        approveRequest();
        visitApprovedDeferralsTab();

        cy.get('table tr:nth(1) td[data-label="Request name"] a').then((element) => {
            const requestName = element.text().trim();
            typeAndEnterCustomSearchFilterValue('Exception', 'Request Name', requestName);
            cy.get('table tr:nth(1) td[data-label="Request name"] a').should('exist');
            cy.get(vulnSelectors.clearFiltersButton).click();
            typeAndEnterCustomSearchFilterValue('Exception', 'Request Name', 'BLAH');
            cy.get('table tr:nth(1) td[data-label="Request name"] a').should('not.exist');
        });
    });

    it('should be able to filter by "Requester"', () => {
        deferAndVisitRequestDetails({
            comment,
            expiry,
            scope,
        });
        approveRequest();
        visitApprovedDeferralsTab();

        typeAndEnterCustomSearchFilterValue('Exception', 'Requester User Name', 'ui_tests');
        cy.get('table tr:nth(1) td[data-label="Request name"] a').should('exist');
        cy.get(vulnSelectors.clearFiltersButton).click();
        typeAndEnterCustomSearchFilterValue('Exception', 'Requester User Name', 'BLAH');
        cy.get('table tr:nth(1) td[data-label="Request name"] a').should('not.exist');
    });
});
