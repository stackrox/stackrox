import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import {
    cancelAllCveExceptions,
    fillAndSubmitExceptionForm,
    selectSingleCveForException,
    verifyExceptionConfirmationDetails,
    verifySelectedCvesInModal,
    visitWorkloadCveOverview,
    typeAndSelectCustomSearchFilterValue,
} from '../workloadCves/WorkloadCves.helpers';
import { visitExceptionManagement } from './ExceptionManagement.helpers';
import { selectors } from './ExceptionManagement.selectors';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';

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
        cy.get(vulnSelectors.clearFiltersButton).click(); // Note: This is a workaround to prevent a lack of CVE data from causing the test to fail in CI

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
        cy.get(vulnSelectors.clearFiltersButton).click(); // Note: This is a workaround to prevent a lack of CVE data from causing the test to fail in CI

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
        cy.get(vulnSelectors.clearFiltersButton).click(); // Note: This is a workaround to prevent a lack of CVE data from causing the test to fail in CI

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

    it('should be able to sort on the "Request Name" column', () => {
        visitExceptionManagement();

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
        visitExceptionManagement();

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
        visitExceptionManagement();

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
        visitExceptionManagement();

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
        visitExceptionManagement();

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
        cy.get(vulnSelectors.clearFiltersButton).click(); // Note: This is a workaround to prevent a lack of CVE data from causing the test to fail in CI

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

            cy.get('table td[data-label="Request name"] a').then((element) => {
                const requestName = element.text().trim();
                typeAndSelectCustomSearchFilterValue('Request name', requestName);
                cy.get('table td[data-label="Request name"] a').should('exist');
            });
        });
    });

    it('should be able to filter by "Requester"', () => {
        visitWorkloadCveOverview();
        cy.get(vulnSelectors.clearFiltersButton).click(); // Note: This is a workaround to prevent a lack of CVE data from causing the test to fail in CI

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

            typeAndSelectCustomSearchFilterValue('Requester', 'ui_tests');
            cy.get('table td[data-label="Request name"] a').should('exist');
            cy.get(vulnSelectors.clearFiltersButton).click();
            typeAndSelectCustomSearchFilterValue('Requester', 'BLAH');
            cy.get('table td[data-label="Request name"] a').should('not.exist');
        });
    });
});
