import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';

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

describe('Workload CVE List deferral and false positive flows', () => {
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
        cy.get(selectors.paginationNext).click();
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
        cy.get(selectors.paginationPrevious).click();
        cy.get(selectors.isUpdatingTable).should('not.exist');
        cy.get(selectors.bulkActionMenuToggle).should('be.disabled');
        cy.get(selectors.tableRowSelectAllCheckbox).should('be.disabled');
        cy.get(selectors.tableRowSelectCheckbox).should('not.be.checked');
    });

    it('should defer a single CVE', () => {
        visitWorkloadCveOverview();
        cy.get(selectors.clearFiltersButton).click(); // Note: This is a workaround to prevent a lack of CVE data from causing the test to fail in CI

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
        cy.get(selectors.clearFiltersButton).click(); // Note: This is a workaround to prevent a lack of CVE data from causing the test to fail in CI

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

    it('should mark a single CVE false positive', () => {
        visitWorkloadCveOverview();
        cy.get(selectors.clearFiltersButton).click(); // Note: This is a workaround to prevent a lack of CVE data from causing the test to fail in CI

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
        cy.get(selectors.clearFiltersButton).click(); // Note: This is a workaround to prevent a lack of CVE data from causing the test to fail in CI

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
