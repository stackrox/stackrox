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

    it('should disable multi-cve controls when no rows are selected', () => {
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

        selectSingleCveForException('DEFERRAL').then((cveName) => {
            verifySelectedCvesInModal([cveName]);
            fillAndSubmitExceptionForm('Test comment', 'When all CVEs are fixable');
            verifyExceptionConfirmationDetails(
                'Deferral',
                [cveName],
                'All images',
                'When all CVEs are fixable'
            );
        });
    });

    it('should defer multiple selected CVEs', () => {
        visitWorkloadCveOverview();

        selectMultipleCvesForException('DEFERRAL').then((cveNames) => {
            verifySelectedCvesInModal(cveNames);
            fillAndSubmitExceptionForm('Test comment', '30 days');

            verifyExceptionConfirmationDetails(
                'Deferral',
                cveNames,
                'All images',
                `${getDateString(getFutureDateByDays(30))} (30 days)`
            );
        });
    });

    it('should mark a single CVE false positive', () => {
        visitWorkloadCveOverview();

        selectSingleCveForException('FALSE_POSITIVE').then((cveName) => {
            verifySelectedCvesInModal([cveName]);
            fillAndSubmitExceptionForm('Test comment');
            verifyExceptionConfirmationDetails('False positive', [cveName], 'All images');
        });
    });

    it('should mark multiple selected CVEs as false positive', () => {
        visitWorkloadCveOverview();

        selectMultipleCvesForException('FALSE_POSITIVE').then((cveNames) => {
            verifySelectedCvesInModal(cveNames);
            fillAndSubmitExceptionForm('Test comment');
            verifyExceptionConfirmationDetails('False positive', cveNames, 'All images');
        });
    });
});
