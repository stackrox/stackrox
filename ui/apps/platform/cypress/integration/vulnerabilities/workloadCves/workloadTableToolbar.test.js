import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import { visitWorkloadCveOverview } from './WorkloadCves.helpers';
import { selectors } from './WorkloadCves.selectors';

describe('Workload table toolbar', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES')) {
            this.skip();
        }
    });

    // TODO Fix the flake and re-enable this test https://issues.redhat.com/browse/ROX-17959
    it.skip('should correctly handle applied filters', () => {
        visitWorkloadCveOverview();

        // Set the entity type to 'Namespace'
        cy.get(selectors.searchOptionsDropdown).click();
        cy.get(selectors.searchOptionsMenuItem('Namespace')).click();
        cy.get(selectors.searchOptionsDropdown).click();
        cy.get(selectors.searchOptionsDropdown).should('have.text', 'Namespace');

        // Apply a namespace filter
        cy.get(selectors.searchOptionsValueTypeahead('Namespace')).click();
        cy.get(selectors.searchOptionsValueTypeahead('Namespace')).type('stackrox');
        cy.get(selectors.searchOptionsValueMenuItem('Namespace')).contains('stackrox').click();

        cy.get(selectors.searchOptionsValueTypeahead('Namespace')).click();

        // Apply a severity filter
        cy.get(selectors.severityDropdown).click();
        cy.get(selectors.severityMenuItem('Critical')).click();
        cy.get(selectors.severityMenuItem('Important')).click();
        cy.get(selectors.severityDropdown).click();

        // Check that the filters are applied in the toolbar chips
        cy.get(selectors.filterChipGroupItem('Namespace', 'stackrox'));
        cy.get(selectors.filterChipGroupItem('Severity', 'Critical'));
        cy.get(selectors.filterChipGroupItem('Severity', 'Important'));
        cy.get(selectors.filterChipGroupItem('Severity', 'Moderate')).should('not.exist');

        // Test removing filters
        cy.get(selectors.filterChipGroupItemRemove('Severity', 'Important')).click();
        cy.get(selectors.filterChipGroupItem('Severity', 'Important')).should('not.exist');

        // Test that changing to the deployment entity tab persists the filters
        cy.get(selectors.entityTypeToggleItem('Deployment')).click();
        cy.get(selectors.filterChipGroupItem('Namespace', 'stackrox'));
        cy.get(selectors.filterChipGroupItem('Severity', 'Critical'));

        // Clear remaining filters
        cy.get(selectors.clearFiltersButton).click();
        cy.get(selectors.filterChipGroup('Severity')).should('not.exist');
        cy.get(selectors.filterChipGroup('Namespace')).should('not.exist');
    });
});
