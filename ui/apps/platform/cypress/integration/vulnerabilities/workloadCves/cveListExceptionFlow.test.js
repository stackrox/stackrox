import withAuth from '../../../helpers/basicAuth';

import {
    cancelAllCveExceptions,
    fillAndSubmitExceptionForm,
    getDateString,
    getFutureDateByDays,
    selectMultipleCvesForException,
    selectSingleCveForException,
    verifyExceptionConfirmationDetails,
    verifySelectedCvesInModal,
    visitWorkloadCveOverview,
} from './WorkloadCves.helpers';
import { selectors } from './WorkloadCves.selectors';
import { paginateNext, paginatePrevious } from '../../../helpers/tableHelpers';

describe('Workload CVE List deferral and false positive flows', () => {
    withAuth();

    beforeEach(() => {
        cancelAllCveExceptions();
    });

    after(() => {
        cancelAllCveExceptions();
    });

    // TODO - Update this test to mock the server response since we can't rely on multiple pages of data
    it.skip('should disable multi-cve controls when no rows are selected', () => {
        visitWorkloadCveOverview();
        // Check that the select all checkbox is disabled
        // Check that the bulk action menu is disabled
        cy.get(selectors.tableRowSelectAllCheckbox).should('be.disabled');
        cy.get(selectors.bulkActionMenuToggle).should('be.disabled');

        // Select a single row and
        // - check that the select all checkbox is enabled
        // - check that the bulk action menu is enabled
        cy.get(selectors.firstTableRow).find(selectors.tableRowSelectCheckbox).click();
        cy.get(selectors.tableRowSelectAllCheckbox).should('not.be.disabled');
        cy.get(selectors.bulkActionMenuToggle).should('not.be.disabled');

        // Move to the next page and
        // - check that the select all checkbox is enabled
        // - check that the bulk action menu is enabled
        paginateNext();
        cy.get(selectors.isUpdatingTable).should('not.exist');
        cy.get(selectors.tableRowSelectAllCheckbox).should('not.be.disabled');
        cy.get(selectors.bulkActionMenuToggle).should('not.be.disabled');

        // Click the select all checkbox and
        // - check that the bulk action menu is disabled
        // - check that the select all checkbox is disabled
        cy.get(selectors.tableRowSelectAllCheckbox).click();
        cy.get(selectors.bulkActionMenuToggle).should('be.disabled');
        cy.get(selectors.tableRowSelectAllCheckbox).should('be.disabled');

        // Move to the previous page and
        // - check that the bulk action menu is disabled
        // - check that the select all checkbox is disabled
        // - check that no table rows are checked
        paginatePrevious();
        cy.get(selectors.isUpdatingTable).should('not.exist');
        cy.get(selectors.bulkActionMenuToggle).should('be.disabled');
        cy.get(selectors.tableRowSelectAllCheckbox).should('be.disabled');
        cy.get(selectors.tableRowSelectCheckbox).should('not.be.checked');
    });

    it('should defer a single CVE', () => {
        visitWorkloadCveOverview();

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
        });
    });

    it('should defer multiple selected CVEs', () => {
        visitWorkloadCveOverview();

        selectMultipleCvesForException('DEFERRAL').then((cveNames) => {
            verifySelectedCvesInModal(cveNames);
            fillAndSubmitExceptionForm({ comment: 'Test comment', expiryLabel: '30 days' });

            verifyExceptionConfirmationDetails({
                expectedAction: 'Deferral',
                cves: cveNames,
                scope: 'All images',
                expiry: `${getDateString(getFutureDateByDays(30))} (30 days)`,
            });
        });
    });

    it('should allow selecting multiple CVEs but enable undoing some selections in the exception modal', () => {
        visitWorkloadCveOverview();

        selectMultipleCvesForException('DEFERRAL').then((cveNames) => {
            const cveToUnselect = cveNames[0];

            verifySelectedCvesInModal(cveNames);

            // unselect the first CVE
            cy.get(`*[role="dialog"] button[aria-label="Remove ${cveToUnselect}"]`).click();

            fillAndSubmitExceptionForm({ comment: 'Test comment', expiryLabel: '30 days' });

            verifyExceptionConfirmationDetails({
                expectedAction: 'Deferral',
                cves: cveNames.filter((cve) => cve !== cveToUnselect), // The first CVE should have been unselected
                scope: 'All images',
                expiry: `${getDateString(getFutureDateByDays(30))} (30 days)`,
            });
        });
    });

    it('should mark a single CVE false positive', () => {
        visitWorkloadCveOverview();

        selectSingleCveForException('FALSE_POSITIVE').then((cveName) => {
            verifySelectedCvesInModal([cveName]);
            fillAndSubmitExceptionForm({ comment: 'Test comment' });
            verifyExceptionConfirmationDetails({
                expectedAction: 'False positive',
                cves: [cveName],
                scope: 'All images',
            });
        });
    });

    it('should mark multiple selected CVEs as false positive', () => {
        visitWorkloadCveOverview();

        selectMultipleCvesForException('FALSE_POSITIVE').then((cveNames) => {
            verifySelectedCvesInModal(cveNames);
            fillAndSubmitExceptionForm({ comment: 'Test comment' });
            verifyExceptionConfirmationDetails({
                expectedAction: 'False positive',
                cves: cveNames,
                scope: 'All images',
            });
        });
    });
});
