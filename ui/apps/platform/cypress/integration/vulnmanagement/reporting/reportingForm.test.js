import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import { getHelperElementByLabel, getInputByLabel } from '../../../helpers/formHelpers';

import {
    accessScopesAlias,
    collectionsAlias,
    interactAndVisitVulnerabilityReporting,
    interactAndWaitToCreateReport,
    notifiersAlias,
    visitVulnerabilityReporting,
    visitVulnerabilityReportingToCreate,
} from './reporting.helpers';

describe('Vulnerability Management Reporting form', () => {
    withAuth();

    const isCollectionsEnabled = hasFeatureFlag('ROX_OBJECT_COLLECTIONS');

    it('should navigate from table by button', () => {
        visitVulnerabilityReporting();

        interactAndWaitToCreateReport(
            () => {
                cy.get('a:contains("Create report")').click();
            },
            undefined,
            isCollectionsEnabled
        );

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
        const notifiersAliasMap = {
            [notifiersAlias]: {
                fixture: 'integrations/notifiers.json',
            },
        };
        const accessScopeMap = {
            [accessScopesAlias]: {
                fixture: 'scopes/resourceScopes.json',
            },
        };
        const collectionsMap = {
            [collectionsAlias]: {
                fixture: 'collections/collections.json',
            },
        };
        const staticResponseMap = isCollectionsEnabled
            ? { ...notifiersAliasMap, ...collectionsMap }
            : { ...notifiersAliasMap, ...accessScopeMap };

        visitVulnerabilityReportingToCreate(staticResponseMap, isCollectionsEnabled);

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
});
