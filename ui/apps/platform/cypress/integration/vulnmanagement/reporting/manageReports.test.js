import * as api from '../../../constants/apiEndpoints';
import { url, selectors } from '../../../constants/VulnManagementPage';
import withAuth from '../../../helpers/basicAuth';
import { getHelperElementByLabel, getInputByLabel } from '../../../helpers/formHelpers';
import { visit } from '../../../helpers/visit';
import {
    visitVulnerabilityReporting,
    visitVulnerabilityReportingFromLeftNav,
} from '../../../helpers/vulnmanagement/reporting';
import navigationSelectors from '../../../selectors/navigation';

describe('Vulnmanagement reports', () => {
    withAuth();

    describe('creating a report', () => {
        it('should go from left navigation', () => {
            visitVulnerabilityReportingFromLeftNav();

            cy.location('pathname').should('eq', url.reporting.list);
        });

        it('should go to url and select item in nav bar', () => {
            visitVulnerabilityReporting();

            cy.get(`${navigationSelectors.navExpandable}:contains("Vulnerability Management")`);
            cy.get(`${navigationSelectors.nestedNavLinks}:contains("Reporting")`).should(
                'have.class',
                'pf-m-current'
            );
        });

        it('should navigate to the Create Report view by button', () => {
            visitVulnerabilityReporting();

            cy.intercept('GET', api.accessScopes.list).as('getSimpleAccessScopes');
            cy.intercept('GET', api.integrations.notifiers).as('getNotifiers');
            cy.get(selectors.reportSection.createReportLink).click();
            cy.wait(['@getSimpleAccessScopes', '@getNotifiers']);

            cy.location('pathname').should('eq', url.reporting.list);
            cy.location('search').should('eq', '?action=create');

            cy.get('h1:contains("Create an image vulnerability report")');

            // check the breadcrumbs
            cy.get(selectors.reportSection.breadcrumbItems)
                .last()
                .contains('Create an image vulnerability report');

            // first breadcrumb should be link back to reports table
            cy.get(selectors.reportSection.breadcrumbItems).first().click();
            cy.get('h1:contains("Vulnerability reporting")');
            cy.location('search').should('eq', '');
        });

        it('should navigate to the Create Report view by url', () => {
            cy.intercept('GET', api.accessScopes.list, {
                fixture: 'scopes/resourceScopes.json',
            }).as('getSimpleAccessScopes');
            cy.intercept('GET', api.integrations.notifiers, {
                fixture: 'integrations/notifiers.json',
            }).as('getNotifiers');
            visit(`${url.reporting.list}?action=create`);
            cy.wait(['@getSimpleAccessScopes', '@getNotifiers']);

            cy.get('h1:contains("Create an image vulnerability report")');

            // Step 0, should start out with disabled Save button
            cy.get(selectors.reportSection.buttons.create).should('be.disabled');

            // Step 1, check empty fields
            getInputByLabel('Report name').type(' ').blur();
            getInputByLabel('Distribution list').focus().blur();

            getHelperElementByLabel('Report name').contains('A report name is required');
            getHelperElementByLabel('Distribution list').contains(
                'At least one email address is required'
            );
            cy.get(selectors.reportSection.buttons.create).should('be.disabled');

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

            cy.get(selectors.reportSection.buttons.create).should('be.disabled');

            // Step 3, check valid from and save
            getInputByLabel('Distribution list')
                .clear()
                .type('scooby@mysteryinc.com,shaggy@mysteryinc.com', {
                    parseSpecialCharSequences: false,
                })
                .blur();

            // TODO: once we are able to manipulate the PatternFly select element, uncomment and complete the test
            // cy.get(selectors.reportSection.buttons.create).should('be.enabled').click();
        });
    });
});
