import withAuth from '../../../helpers/basicAuth';
import {
    cancelAllCveExceptions,
    typeAndEnterCustomSearchFilterValue,
    viewCvesByObservationState,
    visitWorkloadCveOverview,
} from '../workloadCves/WorkloadCves.helpers';
import {
    markFalsePositiveAndVisitRequestDetails,
    visitApprovedFalsePositivesTab,
    approveRequest,
} from './ExceptionManagement.helpers';
import { selectors } from './ExceptionManagement.selectors';
import { selectors as workloadSelectors } from '../workloadCves/WorkloadCves.selectors';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';

const comment = 'False positive!';
const scope = 'All images';

describe('Exception Management - Approved False Positives Table', () => {
    withAuth();

    beforeEach(() => {
        cancelAllCveExceptions();
    });

    after(() => {
        cancelAllCveExceptions();
    });

    it('should be able to view approved false positives', () => {
        markFalsePositiveAndVisitRequestDetails({
            comment,
            scope,
        });
        approveRequest();
        visitApprovedFalsePositivesTab();

        // the deferred request should be approved
        cy.get(
            'table tr:nth(1) td[data-label="Requested action"]:contains("False positive")'
        ).should('exist');
    });

    it('should navigate from Workload CVEs to a request list filtered by the specific CVE', () => {
        markFalsePositiveAndVisitRequestDetails({ comment, scope }).then(
            ({ requestName, cveName }) => {
                approveRequest();

                visitWorkloadCveOverview();
                viewCvesByObservationState('False positives');

                // Verify correct CVE filter
                cy.get('td[data-label="Request details"] a:contains("View")').click();
                cy.get(workloadSelectors.filterChipGroupItem('CVE', cveName));

                // Verify a link in the table containing the request
                cy.get('td a').contains(requestName);
            }
        );
    });

    it('should be able to navigate to the Request Details page by clicking on the request name', () => {
        markFalsePositiveAndVisitRequestDetails({
            comment,
            scope,
        });
        approveRequest();
        visitApprovedFalsePositivesTab();

        const requestNameSelector = 'table tr:nth(1) td[data-label="Request name"] a';

        cy.get(requestNameSelector)
            .invoke('text')
            .then((requestName) => {
                cy.get(requestNameSelector).click();
                cy.get(`h1:contains("${requestName}")`).should('exist');
            });
    });

    it('should be able to sort on the "Request Name" column', () => {
        visitApprovedFalsePositivesTab();

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
        visitApprovedFalsePositivesTab();

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
        visitApprovedFalsePositivesTab();

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

    it('should be able to sort on the "Scope" column', () => {
        visitApprovedFalsePositivesTab();

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
        markFalsePositiveAndVisitRequestDetails({
            comment,
            scope,
        });
        approveRequest();
        visitApprovedFalsePositivesTab();

        cy.get('table tr:nth(1) td[data-label="Request name"] a').then((element) => {
            const requestName = element.text().trim();
            typeAndEnterCustomSearchFilterValue('Exception', 'Request Name', requestName);
            cy.get(vulnSelectors.clearFiltersButton).click();
            typeAndEnterCustomSearchFilterValue('Exception', 'Request Name', 'BLAH');
            cy.get('table tr:nth(1) td[data-label="Request name"] a').should('not.exist');
        });
    });

    it('should be able to filter by "Request name"', () => {
        markFalsePositiveAndVisitRequestDetails({
            comment,
            scope,
        });
        approveRequest();
        visitApprovedFalsePositivesTab();

        typeAndEnterCustomSearchFilterValue('Exception', 'Requester User Name', 'ui_tests');
        cy.get('table tr:nth(1) td[data-label="Request name"] a').should('exist');
        cy.get(vulnSelectors.clearFiltersButton).click();
        typeAndEnterCustomSearchFilterValue('Exception', 'Requester User Name', 'BLAH');
        cy.get('table tr:nth(1) td[data-label="Request name"] a').should('not.exist');
    });
});
