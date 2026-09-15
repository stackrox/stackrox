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
    complianceEnhancedScanConfigsPath,
    visitComplianceEnhancedScanConfigs,
    visitComplianceEnhancedSchedulesFromLeftNav,
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
        cy.get('.pf-v6-c-empty-state__content a:contains("Create scan schedule")').click();
        cy.location('search').should('eq', '?action=create');

        cy.get('.pf-v6-c-wizard__footer button:contains("Cancel")').click();
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
        cy.get('.pf-v6-c-wizard__footer button:contains("Back")').should('be.disabled');

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
        cy.get('body').type('{esc}'); // close the checkbox select before opening TimePicker
        cy.get('input[aria-label="Time picker"]').click(); // PF Datepicker doesn't follow pattern used by helper function
        cy.get('ul[role="menu"] button:contains("00:30")').click();

        // Node roles: defaults to master + worker; add a custom role and remove worker
        cy.get('input[placeholder="Type a role and press Enter to add"]').type('infra{enter}');
        cy.get('.pf-v6-c-label__content:contains("infra")');
        cy.get('.pf-v6-c-label:contains("worker") button[aria-label="Close worker"]').click();
        cy.get('.pf-v6-c-label__content:contains("worker")').should('not.exist');

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
                    nodeRoles: ['master', 'infra'],
                },
                clusters: ['f781e077-fb39-4529-a19d-7a3403e181b2'],
            });
        });
    });

    it('should send @all node role, replacing default roles, when creating a scan config', () => {
        const scheduleName = 'all-nodes-scan';
        const scheduleDescription = 'Scan every node role.';

        visitComplianceEnhancedScanConfigs();

        interceptAndMockComplianceIntegrations(() => {
            cy.get('a:contains("Create scan schedule")').eq(0).click();
        });

        cy.get(`h1:contains("Create scan schedule")`);

        // Step 1, fill required fields
        getInputByLabel('Name').clear().type(scheduleName);
        getInputByLabel('Description').click().type(scheduleDescription).blur();
        getInputByLabel('Frequency').click();
        getSelectOption('Daily').click();
        cy.get('input[aria-label="Time picker"]').click();
        cy.get('ul[role="menu"] button:contains("00:30")').click();

        // Node roles: selecting @all replaces the default master + worker roles
        cy.get('.pf-v6-c-label__content:contains("master")');
        cy.get('.pf-v6-c-label__content:contains("worker")');
        cy.get('input[placeholder="Type a role and press Enter to add"]').type('@all{enter}');
        cy.get('.pf-v6-c-label__content:contains("@all")');
        cy.get('.pf-v6-c-label__content:contains("master")').should('not.exist');
        cy.get('.pf-v6-c-label__content:contains("worker")').should('not.exist');

        navigateWizardNext();

        cy.get('tr:has(td:contains("Healthy")) td input[type="checkbox"]').click();

        interceptAndMockComplianceProfiles(navigateWizardNext);

        cy.get('td input[type="checkbox"]').eq(0).click();

        navigateWizardNext();

        navigateWizardNext();

        interceptAndWaitForCreateScanSchedule(() => {
            cy.get('button:contains("Save")').click();
        }).should(({ request }) => {
            expect(request.body.scanConfig.nodeRoles).to.deep.equal(['@all']);
        });
    });
});
