import withAuth from '../../helpers/basicAuth';
import { readFileFromDownloads } from '../../helpers/file';
import {
    getHelperElementByLabel,
    getInputByLabel,
    generateNameWithDate,
    getSelectButtonByLabel,
    getSelectOption,
} from '../../helpers/formHelpers';

import {
    assertIntegrationsTable,
    cleanupClusterRegistrationSecretsWithName,
    clickCreateNewIntegrationInTable,
    clickIntegrationSourceLinkInForm,
    generateCreatedAuthProvidersIntegrationInForm,
    revokeAuthProvidersIntegrationInTable,
    visitIntegrationsTable,
    visitIntegrationsDashboardFromLeftNav,
} from './integrations.helpers';
import { selectors } from './integrations.selectors';

// Page address segments are the source of truth for integrationSource and integrationType.
const integrationSource = 'authProviders';

describe('Authentication Tokens', () => {
    withAuth();

    it('should create a new API Token and then view and delete', () => {
        const integrationType = 'apitoken';
        const apiTokenName = generateNameWithDate('API Token Test');

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType, 'Generate token');

        // Step 0, should start out with disabled Generate button
        cy.get(selectors.buttons.generate).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Token name').type(' ').blur();

        getHelperElementByLabel('Token name').contains('A token name is required');
        cy.get(selectors.buttons.generate).should('be.disabled');

        // Step 2, check valid from and generate
        getInputByLabel('Token name').clear().type(apiTokenName);
        getSelectButtonByLabel('Role').click();
        getSelectOption('Admin').click();

        generateCreatedAuthProvidersIntegrationInForm(integrationType);

        cy.get('.pf-v5-screen-reader:contains("Success alert")');

        cy.get(selectors.buttons.back).click();

        // View it.

        assertIntegrationsTable(integrationSource, integrationType);

        cy.get(`${selectors.tableRowNameLink}:contains("${apiTokenName}")`).click();

        cy.get(`${selectors.breadcrumbItem}:contains("${apiTokenName}")`);

        clickIntegrationSourceLinkInForm(integrationSource, integrationType);

        // Revoke it.

        revokeAuthProvidersIntegrationInTable(integrationType, apiTokenName);

        cy.get(`${selectors.tableRowNameLink}:contains("${apiTokenName}")`).should('not.exist');
    });

    describe('Cluster registration secrets', () => {
        const testCrsName = 'CYPRESS-e2e-test';

        beforeEach(() => {
            cleanupClusterRegistrationSecretsWithName(testCrsName);
        });

        afterEach(() => {
            cleanupClusterRegistrationSecretsWithName(testCrsName);
        });

        it('should create a new Cluster registration secret and then view and delete', () => {
            visitIntegrationsDashboardFromLeftNav();

            cy.get('section:contains("Authentication Tokens")').scrollIntoView();
            cy.get(
                'section:contains("Authentication Tokens") a:contains("Cluster Registration Secret")'
            ).click();

            const crsLinkInTableSelector = `td[data-label="Name"] a:contains("${testCrsName}")`;

            // Verify that old CRS from Cypress tests are not present
            cy.get('table');
            cy.get(crsLinkInTableSelector).should('not.exist');

            // Create a new CRS and verify that the YAML is downloaded and the secret appears in the table
            // TODO dv 2025-04-01
            // From a user's point of view, this is a "button". It would be nice if we had a unified way to
            // click a button without needing to know the underlying HTML element used by the component.
            cy.get('a:contains("Create cluster registration secret")').click();

            cy.get('button:contains("Download")').should('be.disabled');
            getInputByLabel('Name').clear().type(testCrsName);
            cy.get('button:contains("Download")').click();

            readFileFromDownloads(`${testCrsName}-cluster-registration-secret.yaml`).should(
                'exist'
            );

            // Revoke the secret
            cy.get('table');
            cy.get(`tr:has(${crsLinkInTableSelector}) button[aria-label="Kebab toggle"]`).click();
            // The Revoke button in the table menu
            cy.get('ul[role="menu"] button:contains("Revoke cluster registration secret")').click();
            // The Revoke confirmation button in the modal
            cy.get(
                'div[role="dialog"] button:contains("Revoke cluster registration secret")'
            ).click();

            cy.get('table');
            cy.get(crsLinkInTableSelector).should('not.exist');
        });
    });
});
