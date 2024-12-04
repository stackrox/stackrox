import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';
import { getHelperElementByLabel, getInputByLabel } from '../../helpers/formHelpers';

import {
    visitComplianceEnhancedSchedulesFromLeftNav,
    visitComplianceEnhancedScanConfigs,
    complianceEnhancedScanConfigsPath,
} from './ComplianceEnhanced.helpers';

describe('Compliance Schedules', () => {
    withAuth();

    it('should visit schedules using the left nav', () => {
        visitComplianceEnhancedSchedulesFromLeftNav();

        cy.location('pathname').should('eq', complianceEnhancedScanConfigsPath);
        cy.title().should('match', getRegExpForTitleWithBranding('Cluster compliance'));
    });

    it('should have expected elements on the scan configs page', () => {
        visitComplianceEnhancedScanConfigs();

        cy.title().should('match', getRegExpForTitleWithBranding('Scan schedules'));

        cy.get('th[scope="col"]:contains("Name")');
        cy.get('th[scope="col"]:contains("Schedule")');
        cy.get('th[scope="col"]:contains("Last scanned")');
        cy.get('th[scope="col"]:contains("Clusters")');
        cy.get('th[scope="col"]:contains("Profiles")');

        // check empty state message and call-to-action
        cy.get('h2:contains("No scan schedules")');
        cy.get('.pf-v5-c-empty-state__content a:contains("Create scan schedule")').click();
        cy.location('search').should('eq', '?action=create');

        cy.get('.pf-v5-c-wizard__footer button:contains("Cancel")').click();
    });

    it('should have have a form to add a new scan config', () => {
        visitComplianceEnhancedScanConfigs();

        cy.get('.pf-v5-l-flex.pf-m-row a:contains("Create scan schedule")').click();

        cy.get(`h1:contains("Create scan schedule")`);

        // Step 0, should start out with disabled Back button
        cy.get('.pf-v5-c-wizard__footer button:contains("Back")').should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Name').click().blur();
        getInputByLabel('Frequency').click().click(); // blur with no selection
        cy.get('input[aria-label="Time picker"]').click(); // PF Datepicker doesn't follow pattern used by helper function
        getInputByLabel('Description')
            .click()
            .type('Mare eats oats, and does eat oats, and little lambs eat ivy.')
            .blur();

        getHelperElementByLabel('Name').contains('Name is required');
        getHelperElementByLabel('Time').contains('Time is required');

        getInputByLabel('Frequency').click();
        cy.get('.pf-v5-c-select.pf-m-expanded button:contains("Weekly")').click();
        getInputByLabel('On day(s)').click().click(); // blur with no selection
        getInputByLabel('Name').click();

        getHelperElementByLabel('On day(s)').contains('Selection is required');

        // Step 2, check valid form and save
        getInputByLabel('Name').clear().type('scooby-doo');
        getInputByLabel('On day(s)').click();
        cy.get('.pf-v5-c-select.pf-m-expanded .pf-v5-c-check__label:contains("Tuesday")').click();
        cy.get('input[aria-label="Time picker"]').click(); // PF Datepicker doesn't follow pattern used by helper function
        cy.get('.pf-v5-c-menu.pf-m-scrollable button:contains("00:30")').click();

        cy.get('.pf-v5-c-wizard__footer button:contains("Next")').click();
    });
});
