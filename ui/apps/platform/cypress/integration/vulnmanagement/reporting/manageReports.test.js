import * as api from '../../../constants/apiEndpoints';
import { url, selectors } from '../../../constants/VulnManagementPage';
import withAuth from '../../../helpers/basicAuth';
import { getHelperElementByLabel, getInputByLabel } from '../../../helpers/formHelpers';

describe('Vulnmanagement reports', () => {
    withAuth();

    describe('creating a report', () => {
        let notifiers;
        let resourcesScopes;

        before(() => {
            cy.fixture('integrations/notifiers.json').then((response) => {
                notifiers = response;
            });
            cy.fixture('scopes/resourceScopes.json').then((response) => {
                resourcesScopes = response;
            });
        });

        beforeEach(() => {
            cy.intercept('GET', api.report.configurations, { reportConfigs: [] }).as(
                'getReportConfigurations'
            );
            cy.intercept('GET', api.report.configurationsCount, { count: 0 }).as(
                'getReportConfigurationsCount'
            );
            cy.intercept('POST', api.graphql('searchOptions')).as('searchOptions');
            cy.intercept('GET', api.integrations.notifiers, notifiers).as('getNotifiers');
            cy.intercept('GET', api.accessScopes.list, resourcesScopes).as('getResourceScopes');
        });

        it('should navigate to the Create Report view by button or directly', () => {
            cy.visit('/main/dashboard');
            cy.get(selectors.vulnManagementExpandableNavLink).click({ force: true });
            cy.get(selectors.vulnManagementExpandedReportingNavLink).click({ force: true });
            cy.url().should('contain', url.reporting.list);

            cy.wait('@getReportConfigurations');
            cy.wait('@getReportConfigurationsCount');

            cy.wait('@searchOptions');

            // Hard-coded wait is to ameliorate a tenacious flake in CI that has resisted all more gentle solutions
            cy.wait(1000);

            cy.get(selectors.reportSection.createReportLink).click();
            cy.location('pathname').should('eq', `${url.reporting.list}`);
            cy.location('search').should('eq', '?action=create');

            // check the breadcrumbs
            cy.get(selectors.reportSection.breadcrumbItems)
                .last()
                .contains('Create a vulnerability report');
            // first breadcrumb should be link back to reports table
            cy.get(selectors.reportSection.breadcrumbItems).first().click();
            cy.location('pathname').should('eq', `${url.reporting.list}`);

            // navigate directly by URL
            cy.visit('/main/dashboard'); // leave Create Report page
            cy.visit(`${url.reporting.list}?action=create`);
            cy.get('h1:contains("Create a vulnerability report")');
        });

        it('should should allow creating a new Report Configuration', () => {
            cy.visit(`${url.reporting.list}?action=create`);

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
