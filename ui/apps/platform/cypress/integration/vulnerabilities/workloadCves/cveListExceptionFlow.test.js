import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';

import {
    selectMultipleCvesForException,
    selectSingleCveForException,
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
        // TODO - clean up any existing deferred or false positive CVEs
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

        selectSingleCveForException('DEFERRAL');
    });

    it('should defer multiple selected CVEs', () => {
        visitWorkloadCveOverview();

        selectMultipleCvesForException('DEFERRAL');
    });

    it('should mark a single CVE false positive', () => {
        visitWorkloadCveOverview();

        selectSingleCveForException('FALSE_POSITIVE');
    });

    it('should mark multiple selected CVEs as false positive', () => {
        visitWorkloadCveOverview();

        selectMultipleCvesForException('FALSE_POSITIVE');
    });
});
