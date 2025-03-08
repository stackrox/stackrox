import withAuth from '../../../helpers/basicAuth';
import {
    cancelAllCveExceptions,
    fillAndSubmitExceptionForm,
    selectSingleCveForException,
    verifyExceptionConfirmationDetails,
    verifySelectedCvesInModal,
    visitWorkloadCveOverview,
    typeAndEnterCustomSearchFilterValue,
} from '../workloadCves/WorkloadCves.helpers';
import { visitPendingRequestsTab } from './ExceptionManagement.helpers';
import { selectors } from './ExceptionManagement.selectors';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';

describe('Exception Management Pending Requests Page', () => {
    withAuth();

    beforeEach(() => {
        cancelAllCveExceptions();
    });

    after(() => {
        cancelAllCveExceptions();
    });

    it('should be able to view deferred pending requests', () => {
        visitWorkloadCveOverview();

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

            visitPendingRequestsTab();

            // the deferred request should be pending
            cy.get(
                'table td[data-label="Requested action"]:contains("Deferred (when all fixed)")'
            ).should('exist');
        });
    });

    it('should be able to view false positive pending requests', () => {
        visitWorkloadCveOverview();

        // mark a single cve as false positive
        selectSingleCveForException('FALSE_POSITIVE').then((cveName) => {
            verifySelectedCvesInModal([cveName]);
            fillAndSubmitExceptionForm({ comment: 'Test comment' });
            verifyExceptionConfirmationDetails({
                expectedAction: 'False positive',
                cves: [cveName],
                scope: 'All images',
            });

            visitPendingRequestsTab();

            // the false positive request should be pending
            cy.get('table td[data-label="Requested action"]:contains("False positive")').should(
                'exist'
            );
        });
    });

    it('should be able to navigate to the Request Details page by clicking on the request name', () => {
        visitWorkloadCveOverview();

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
                visitPendingRequestsTab();

                const requestNameLink = 'table td[data-label="Request name"]';

                cy.get(requestNameLink)
                    .invoke('text')
                    .then((requestName) => {
                        cy.get(requestNameLink).click();
                        cy.get(`h1:contains("${requestName}")`).should('exist');
                    });
            });
    });

    it('should be able to sort on the "Request Name" column', () => {
        visitPendingRequestsTab();

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
        visitPendingRequestsTab();

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
        visitPendingRequestsTab();

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
        visitPendingRequestsTab();

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
        visitPendingRequestsTab();

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
        visitWorkloadCveOverview();

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

            visitPendingRequestsTab();

            cy.get('table td[data-label="Request name"] a').then((element) => {
                const requestName = element.text().trim();
                typeAndEnterCustomSearchFilterValue('Exception', 'Request Name', requestName);
                cy.get('table td[data-label="Request name"] a').should('exist');
            });
        });
    });

    it('should be able to filter by "Requester"', () => {
        visitWorkloadCveOverview();

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

            visitPendingRequestsTab();

            typeAndEnterCustomSearchFilterValue('Exception', 'Requester User Name', 'ui_tests');
            cy.get('table td[data-label="Request name"] a').should('exist');
            cy.get(vulnSelectors.clearFiltersButton).click();
            typeAndEnterCustomSearchFilterValue('Exception', 'Requester User Name', 'BLAH');
            cy.get('table td[data-label="Request name"] a').should('not.exist');
        });
    });
});
