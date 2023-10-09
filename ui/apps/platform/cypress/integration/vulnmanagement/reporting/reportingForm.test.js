import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import {
    getDescriptionListGroup,
    getHelperElementByLabel,
    getInputByLabel,
} from '../../../helpers/formHelpers';
import { tryDeleteCollection } from '../../collections/Collections.helpers';
import { tryDeleteIntegration } from '../../integrations/integrations.helpers';

import {
    collectionsAlias,
    interactAndVisitVulnerabilityReporting,
    interactAndWaitToCreateReport,
    notifiersAlias,
    tryDeleteVMReportConfigs,
    visitVulnerabilityReporting,
    visitVulnerabilityReportingToCreate,
} from './reporting.helpers';

describe('Vulnerability Management Reporting form', () => {
    withAuth();

    before(function () {
        if (hasFeatureFlag('ROX_VULN_MGMT_REPORTING_ENHANCEMENTS')) {
            this.skip();
        }
    });

    it('should navigate from table by button', () => {
        visitVulnerabilityReporting();

        interactAndWaitToCreateReport(() => {
            cy.get('a:contains("Create report")').click();
        });

        cy.location('search').should('eq', '?action=create');

        cy.get('h1:contains("Create an image vulnerability report")');

        // check the breadcrumbs
        cy.get(
            '.pf-c-breadcrumb__item:nth-child(2):contains("Create an image vulnerability report")'
        );

        // first breadcrumb should be link back to reports table
        interactAndVisitVulnerabilityReporting(() => {
            cy.get(
                '.pf-c-breadcrumb__item:nth-child(1) a:contains("Vulnerability reporting")'
            ).click();
        });
        cy.location('search').should('eq', '');
    });

    it('should navigate by url', () => {
        const staticResponseMap = {
            [notifiersAlias]: {
                fixture: 'integrations/notifiers.json',
            },
            [collectionsAlias]: {
                fixture: 'collections/collections.json',
            },
        };

        visitVulnerabilityReportingToCreate(staticResponseMap);

        cy.get('h1:contains("Create an image vulnerability report")');

        // Step 0, should start out with disabled Save button
        cy.get('button:contains("Create")').should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Report name').type(' ').blur();
        getInputByLabel('Distribution list').focus().blur();

        getHelperElementByLabel('Report name').contains('A report name is required');
        getHelperElementByLabel('Distribution list').contains(
            'At least one email address is required'
        );
        cy.get('button:contains("Create")').should('be.disabled');

        // TODO: add checks for FE validation error messages on the following fields
        //       which are not pre-populated
        //       1. On (days to run report)
        //       2. CVE severities
        //
        // Note, the PatternFly select-multiple checkboxes variant does not support
        // Formik blur in a straightforward way, so in order to add tests for lazy
        // validation, we first have to come up with a workaround for that issue

        // Step 2, check fields for invalid formats
        getInputByLabel('Report name').type('Test report 1');
        getInputByLabel('Description').type('A detailed description of the report');

        // TODO: create a method to select from the PatternFly Select elements,
        //       the following does not work
        // getInputByLabel('Configure resource scope').select('UI test scope');

        getInputByLabel('Distribution list')
            .type('scooby,shaggy@mysteryinc.com', {
                parseSpecialCharSequences: false,
            })
            .blur();

        getHelperElementByLabel('Distribution list').contains(
            'List must be valid email addresses, separated by comma'
        );

        cy.get('button:contains("Create")').should('be.disabled');

        // Step 3, check valid from and save
        getInputByLabel('Distribution list')
            .clear()
            .type('scooby@mysteryinc.com,shaggy@mysteryinc.com', {
                parseSpecialCharSequences: false,
            })
            .blur();

        // TODO: once we are able to manipulate the PatternFly select element, uncomment and complete the test
        // cy.get('button:contains("Create")').should('be.enabled').click();
    });

    it('should allow creation of a report configuration', () => {
        const reportName = 'report config -e2e test-';
        const reportDescription = 'ui e2e test report';
        const collectionName = 'stackrox ns collection -e2e test-';
        const emailNotifierName = 'report config email notifier -e2e test-';
        const integrationsSource = 'notifiers';

        // Delete collection from previous calls, if present
        tryDeleteVMReportConfigs(reportName);
        tryDeleteCollection(collectionName);
        tryDeleteIntegration(integrationsSource, emailNotifierName);

        visitVulnerabilityReportingToCreate({});

        cy.get('h1:contains("Create an image vulnerability report")');

        // Fill out report details
        getInputByLabel('Report name').type(reportName);
        getInputByLabel('Description').type(reportDescription);

        getInputByLabel('Repeat report…').click();
        cy.get('button[role="option"]:contains("Monthly")').click();

        cy.get('button:has(*:contains("Select days"))').click();
        cy.get('*[role="listbox"] span:contains("The middle of the month")').click();

        getInputByLabel('Distribution list').type('scooby@mysteryinc.com');

        cy.get('button:has(*:contains("Fixable states selected"))').click();
        cy.get('*[role="listbox"] span:contains("Unfixable")').click();

        cy.get('button:has(*:contains("Severities selected"))').click();
        cy.get('*[role="listbox"] span:contains("Moderate")').click();
        cy.get('*[role="listbox"] span:contains("Low")').click();

        // Create an email notifier via modal
        cy.get('button:contains("Create email notifier")').click();
        getInputByLabel('Integration name').type(emailNotifierName);
        getInputByLabel('Email server').type('e2e.test.rox.systems:465');
        getInputByLabel('Enable unauthenticated SMTP').click();
        getInputByLabel('Sender').type('e2e-test-sender@rox.systems');
        getInputByLabel('Default recipient').type('e2e-test-recipient@rox.systems');
        cy.get('*[role="dialog"] button:contains("Save integration")').click();

        // The newly created notifier should automatically be selected in the input
        cy.get(`button[aria-label="Select a notifier"]:contains("${emailNotifierName}")`);

        // Create a collection via modal
        cy.get('button:contains("Create collection")').click();

        cy.get('*[role="dialog"] input[name="name"]').type(collectionName);
        cy.get('*[role="dialog"] button:contains("All namespaces")').click();
        cy.get('*[role="dialog"] button:contains("Namespaces with names matching")').click();
        cy.get(
            '*[role="dialog"] input[aria-label="Select value 1 of 1 for the namespace name"]'
        ).type('stackrox');
        cy.get('*[role="dialog"] button:contains("Save")').click();

        // The newly created collection should automatically be selected
        cy.get(`input[placeholder="Select a collection"]`)
            .invoke('val')
            .should('equal', collectionName);

        // Create the VM Report config
        cy.get('button[data-testid="create-btn"]').click();

        // Redirected back to report config table
        // TODO For some reason when submitting the form via a browser automated by Cypress, we get a white screen with
        // no error information, so we have to manually visit the report config table from here.
        visitVulnerabilityReporting();

        // Find the report in the table and click it to go to the details page
        cy.get(`td a:contains("${reportName}")`).click();

        cy.get(`h1:contains("${reportName}")`);

        // Check that entered values were saved correctly on the details view
        getDescriptionListGroup('Description', reportDescription);
        getDescriptionListGroup('CVE fixability type', 'Fixable');
        getDescriptionListGroup('Notification method', emailNotifierName);
        getDescriptionListGroup('Distribution list', 'scooby@mysteryinc.com');
        getDescriptionListGroup(
            'Reporting schedule',
            'Repeat report monthly on the middle of the month'
        );
        getDescriptionListGroup('CVE severities', 'Critical');
        getDescriptionListGroup('CVE severities', 'Important');
        getDescriptionListGroup('CVE severities', 'Moderate').should('not.exist');
        getDescriptionListGroup('CVE severities', 'Low').should('not.exist');
        getDescriptionListGroup('Report scope', collectionName);

        // Visit the linked collection page
        cy.get(`a:contains("${collectionName}")`).click();

        // Verify that we have landed on the correct collection page
        cy.get(`h1:contains("${collectionName}")`);

        // Attempt to delete the collection while it is in use by the VM report config
        cy.get(`button:contains("Actions")`).click();
        cy.get(`button:contains("Delete collection")`).click();
        cy.get('*[role="dialog"] button:contains("Delete")').click();
        cy.get('*:contains("Collection is in use by one or more report configurations")');

        // Self cleanup
        tryDeleteVMReportConfigs(reportName);
        tryDeleteCollection(collectionName);
        tryDeleteIntegration(integrationsSource, emailNotifierName);
    });
});
