import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { getRegExpForTitleWithBranding } from '../../helpers/title';
import { getHelperElementByLabel, getInputByLabel } from '../../helpers/formHelpers';

import {
    visitComplianceEnhancedClusterComplianceFromLeftNav,
    visitComplianceEnhancedScanConfigs,
    visitComplianceEnhancedClusterCompliance,
    clusterComplianceCoveragePath,
    clusterComplianceScanConfigsPath,
} from './ComplianceEnhanced.helpers';

describe('Compliance Dashboard', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_COMPLIANCE_ENHANCEMENTS')) {
            this.skip();
        }
    });

    it('should visit cluster compliance from the left nav and default to coverage view', () => {
        visitComplianceEnhancedClusterComplianceFromLeftNav();

        cy.location('pathname').should('eq', clusterComplianceCoveragePath);
        cy.title().should('match', getRegExpForTitleWithBranding('Cluster compliance'));
    });

    it('should visit scan configurations scheduling from tab nav menu', () => {
        visitComplianceEnhancedClusterCompliance();
        cy.get('a.pf-c-nav__link').contains('Schedules').click();

        cy.location('pathname').should('eq', clusterComplianceScanConfigsPath);
        cy.get('a.pf-c-nav__link').contains('Schedules').should('have.class', 'pf-m-current');
    });

    it('should have expected elements on the scan configs page', () => {
        visitComplianceEnhancedScanConfigs();

        cy.title().should('match', getRegExpForTitleWithBranding('Scan schedules'));

        cy.get('th[scope="col"]:contains("Name")');
        cy.get('th[scope="col"]:contains("Schedule")');
        cy.get('th[scope="col"]:contains("Last run")');
        cy.get('th[scope="col"]:contains("Clusters")');
        cy.get('th[scope="col"]:contains("Profiles")');

        // check empty state message and call-to-action
        cy.get('h2:contains("No scan schedules")');
        cy.get('.pf-c-empty-state__content a:contains("Create scan schedule")').click();
        cy.location('search').should('eq', '?action=create');

        cy.get('.pf-c-wizard__footer button:contains("Cancel")').click();
    });

    it('should have have a form to add a new scan config', () => {
        visitComplianceEnhancedScanConfigs();

        cy.get('.pf-c-toolbar__content a:contains("Create scan schedule")').click();

        cy.get(`h1:contains("Create scan schedule")`);

        // Step 0, should start out with disabled Back button
        cy.get('.pf-c-wizard__footer button:contains("Back")').should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Name').click().blur();
        getInputByLabel('Frequency').click().click(); // blur with no selection
        cy.get('input[aria-label="Time picker"]').click(); // PF Datepicker doesn't follow pattern used by helper function
        getInputByLabel('Description')
            .click()
            .type('Mare eats oats, and does eat oats, and little lambs eat ivy.')
            .blur();

        getHelperElementByLabel('Name').contains('Scan name is required');
        getHelperElementByLabel('Time').contains('Time is required');

        getInputByLabel('Frequency').click();
        cy.get('.pf-c-select.pf-m-expanded button:contains("Weekly")').click();
        getInputByLabel('On day(s)').click().click(); // blur with no selection
        getInputByLabel('Name').click();

        getHelperElementByLabel('On day(s)').contains('Selection is required');

        // Step 2, check valid form and save
        getInputByLabel('Name').clear().type('scooby-doo');
        getInputByLabel('On day(s)').click();
        cy.get('.pf-c-select.pf-m-expanded .pf-c-check__label:contains("Tuesday")').click();
        cy.get('input[aria-label="Time picker"]').click(); // PF Datepicker doesn't follow pattern used by helper function
        cy.get('.pf-c-menu.pf-m-scrollable button:contains("12:30 AM")').click();

        cy.get('.pf-c-wizard__footer button:contains("Next")').click();
    });
});
