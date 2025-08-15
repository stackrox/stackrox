import withAuth from '../../helpers/basicAuth';
import { interactAndWaitForResponses } from '../../helpers/request';
import { getRegExpForTitleWithBranding } from '../../helpers/title';
import {
    getHelperElementByLabel,
    getInputByLabel,
    getSelectOption,
} from '../../helpers/formHelpers';
import { navigateWizardNext } from '../../helpers/wizard';

import {
    visitComplianceEnhancedSchedulesFromLeftNav,
    visitComplianceEnhancedScanConfigs,
    complianceEnhancedScanConfigsPath,
} from './ComplianceEnhanced.helpers';

function interceptAndMockComplianceIntegrations(callback) {
    const alias = 'compliance/integrations';
    return interactAndWaitForResponses(
        callback,
        {
            [alias]: { method: 'GET', url: '/v2/compliance/integrations' },
        },
        {
            [alias]: { fixture: 'compliance/integrations' },
        }
    );
}

function interceptAndMockComplianceProfiles(callback) {
    const alias = 'compliance/profiles/summary';
    return interactAndWaitForResponses(
        callback,
        {
            [alias]: { method: 'GET', url: '/v2/compliance/profiles/summary?*' },
        },
        {
            [alias]: { fixture: 'compliance/profiles' },
        }
    );
}

function interceptAndWaitForCreateScanSchedule(interactionCallback) {
    cy.intercept('POST', '/v2/compliance/scan/configurations', (req) => {
        req.reply({});
    }).as('createScanSchedule');

    interactionCallback();

    // should filter using the correct values for the "Platform view"
    return cy.wait('@createScanSchedule');
}

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
        const scheduleName = 'scooby-doo';
        const scheduleDescription = 'Mare eats oats, and does eat oats, and little lambs eat ivy.';

        visitComplianceEnhancedScanConfigs();

        interceptAndMockComplianceIntegrations(() => {
            cy.get('a:contains("Create scan schedule")').eq(0).click();
        });

        cy.get(`h1:contains("Create scan schedule")`);

        // Step 0, should start out with disabled Back button
        cy.get('.pf-v5-c-wizard__footer button:contains("Back")').should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Name').click().blur();
        getInputByLabel('Frequency').click().click(); // blur with no selection
        cy.get('input[aria-label="Time picker"]').click(); // PF Datepicker doesn't follow pattern used by helper function
        getInputByLabel('Description').click().type(scheduleDescription).blur();

        getHelperElementByLabel('Name').contains('Name is required');
        getHelperElementByLabel('Time').contains('Time is required');

        getInputByLabel('Frequency').click();
        getSelectOption('Weekly').click();
        getInputByLabel('On day(s)').click().click(); // blur with no selection
        getInputByLabel('Name').click();

        getHelperElementByLabel('On day(s)').contains('Selection is required');

        // Step 2, check valid form and save
        getInputByLabel('Name').clear().type(scheduleName);
        getInputByLabel('On day(s)').click();
        getSelectOption('Tuesday').click();
        cy.get('input[aria-label="Time picker"]').click(); // PF Datepicker doesn't follow pattern used by helper function
        cy.get('ul[role="menu"] button:contains("00:30")').click();

        navigateWizardNext();

        cy.get('tr:has(td:contains("Healthy")) td input[type="checkbox"]').click();

        interceptAndMockComplianceProfiles(navigateWizardNext);

        // Select the first profile
        cy.get('td input[type="checkbox"]').eq(0).click();

        // TODO Skip adding a delivery destination for now
        navigateWizardNext();

        navigateWizardNext();

        interceptAndWaitForCreateScanSchedule(() => {
            cy.get('button:contains("Save")').click();
        }).should(({ request }) => {
            expect(request.body).to.deep.equal({
                scanName: scheduleName,
                scanConfig: {
                    description: scheduleDescription,
                    oneTimeScan: false,
                    profiles: ['CYPRESS-ocp4-bsi'],
                    scanSchedule: {
                        daysOfWeek: { days: [2] },
                        hour: 0,
                        minute: 30,
                        intervalType: 'WEEKLY',
                    },
                    notifiers: [],
                },
                clusters: ['f781e077-fb39-4529-a19d-7a3403e181b2'],
            });
        });
    });
});
